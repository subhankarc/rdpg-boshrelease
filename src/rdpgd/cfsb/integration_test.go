package cfsb_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/lib/pq"

	consulapi "github.com/hashicorp/consul/api"

	// "github.com/starkandwayne/rdpgd/cfsb"
	. "github.com/smartystreets/goconvey/convey"
	"github.com/starkandwayne/rdpgd/cfsb"
	"github.com/starkandwayne/rdpgd/instances"
	"github.com/starkandwayne/rdpgd/pg"
	"github.com/starkandwayne/rdpgd/rdpg"
	"github.com/starkandwayne/rdpgd/uuid"
)

func init() {
	os.Setenv("RDPGD_PB_PORT", "7432")
}

func cfsbAPIURL(path string) string {
	return fmt.Sprintf(`http://cfadmin:cfadmin@10.244.2.2:8888%s`, path)
}

func TestCFSBAPIAuthorization(t *testing.T) {
	Convey("CFSB API Authorization", t, func() {
		// API Version Header is set test
		// basic_auth test, need username and password (Authentication :header) to do broker registrations
		// return 401 Unauthorized if credentials are not valid  test, auth only tested here
		// test when reject a request, response a 412 Precondition Failed message
		var getBasicAuthTests = []struct {
			username, password string
			status             int
		}{
			{"cfadmin", "cfadmin", 200},
			{"Aladdin", "open:sesame", 401},
			{"", "", 401},
			{"cf", "bala", 401},
			{"", "cf", 401},
		}

		for _, authTest := range getBasicAuthTests {
			req, err := http.NewRequest("GET", cfsbAPIURL(`/v2/catalog`), nil)
			So(err, ShouldBeNil)
			req.SetBasicAuth(authTest.username, authTest.password)
			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, http.StatusOK)
		}
	})
}
func TestIntegration(t *testing.T) {
	// Complete integration test,
	// get catalg
	// use results to provision instance
	// user results to bind
	// user results to unbind
	// user results to deprovision
	// - check for ineffective_at timestamp set
	Convey("Integration Worflow", t, func() {

		config := consulapi.DefaultConfig()
		config.Address = `10.244.2.2:8500`
		consulClient, err := consulapi.NewClient(config)
		So(err, ShouldBeNil)

		organizationID := uuid.NewUUID().String()
		spaceID := uuid.NewUUID().String()
		Convey("Get Catalog", func() {
			req, err := http.NewRequest("GET", cfsbAPIURL(`/v2/catalog`), nil)
			So(err, ShouldBeNil)
			req.SetBasicAuth("cfadmin", "cfadmin")

			httpClient := &http.Client{}
			resp, err := httpClient.Do(req)
			So(err, ShouldBeNil)
			So(resp.StatusCode, ShouldEqual, http.StatusOK)

			decoder := json.NewDecoder(resp.Body)
			var c cfsb.Catalog
			err = decoder.Decode(&c)
			So(err, ShouldBeNil)

			// fetch catalog
			So(len(c.Services), ShouldNotEqual, 0)
			Convey("The first service", func() {
				firstService := c.Services[0]
				So(firstService.ServiceID, ShouldNotBeBlank)
				So(firstService.Name, ShouldNotBeBlank)
				So(firstService.Description, ShouldNotBeBlank)
				So(len(firstService.Plans), ShouldNotEqual, 0)
				Convey("The first plan", func() {
					firstPlan := firstService.Plans[0]
					So(firstPlan.PlanID, ShouldNotBeBlank)
					So(firstPlan.Name, ShouldNotBeBlank)
					So(firstPlan.Description, ShouldNotBeBlank)
				})
			})

			serviceID := c.Services[0].ServiceID
			planID := c.Services[0].Plans[0].PlanID
			Convey("Provision Instance", func() {

				type InstanceBody struct {
					ServiceID      string `json:"service_id"`
					PlanID         string `json:"plan_id"`
					OrganizationID string `json:"organization_guid"`
					SpaceID        string `json:"space_guid"`
				}

				ins := &InstanceBody{
					ServiceID:      serviceID,
					PlanID:         planID,
					OrganizationID: organizationID,
					SpaceID:        spaceID,
				}

				instanceID := uuid.NewUUID().String()
				insbody, err := json.Marshal(ins)
				So(err, ShouldBeNil)
				url := cfsbAPIURL("/v2/service_instances/" + instanceID)
				req, err := http.NewRequest("PUT", url, bytes.NewBuffer(insbody))
				req.SetBasicAuth("cfadmin", "cfadmin")
				So(err, ShouldBeNil)

				httpClient := &http.Client{}
				resp, err := httpClient.Do(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)

				// At this point the instance within the database should have the clusterID
				// Note that this will be fetching from the management cluster database
				p := pg.NewPG(`10.244.2.2`, `7432`, `rdpg`, `rdpg`, `admin`)
				db, err := p.Connect()
				So(err, ShouldBeNil)
				i := instances.Instance{}
				sq := fmt.Sprintf(`SELECT id, cluster_id,instance_id, service_id, plan_id, organization_id, space_id, dbname, dbuser, dbpass FROM cfsb.instances WHERE instance_id=lower('%s') LIMIT 1`, instanceID)
				err = db.Get(&i, sq)
				So(err, ShouldBeNil)

				managementCluster, err := rdpg.NewCluster(`rdpgmc`, consulClient)
				So(err, ShouldBeNil)
				Convey("Each management cluster node should have the correct fields set in the cfsb.instances table", func() {
					for _, node := range managementCluster.Nodes { // Loop over management cluster nodes.
						p := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
						db, err := p.Connect()
						So(err, ShouldBeNil)

						in := instances.Instance{}
						sq := fmt.Sprintf(`SELECT id, cluster_id,instance_id, service_id, plan_id, organization_id, space_id, dbname, dbuser, dbpass FROM cfsb.instances WHERE instance_id=lower('%s') LIMIT 1`, instanceID)
						err = db.Get(&in, sq)
						So(err, ShouldBeNil)
						So(in.ClusterID, ShouldNotBeBlank)
						So(in.ServiceID, ShouldEqual, ins.ServiceID)
						So(in.PlanID, ShouldEqual, ins.PlanID)
						So(in.OrganizationID, ShouldEqual, ins.OrganizationID)
						So(in.SpaceID, ShouldEqual, ins.SpaceID)
						db.Close()
					}
				})

				// TODO: Find out which SC the instance is on and for that cluster's nodes, do the below.
				serviceCluster, err := rdpg.NewCluster(i.ClusterID, consulClient)
				So(err, ShouldBeNil)

				var s string
				Convey("Each service cluster node should have the user and database created on it", func() {
					for _, node := range serviceCluster.Nodes { // Loop over service cluster nodes.
						sp := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
						db, err := sp.Connect()
						So(err, ShouldBeNil)

						// user should be created on each service cluster node
						q := fmt.Sprintf(`SELECT rolname FROM pg_roles WHERE rolname='%s'`, i.User)
						err = db.Get(&s, q)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, i.User)

						// database should be created on each service cluster node
						q = fmt.Sprintf(`SELECT datname FROM pg_catalog.pg_database WHERE datname='%s'`, i.Database)
						err = db.Get(&s, q)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, i.Database)
						db.Close()

					}
				})

				Convey("Each node should have a record of the instance in rdpg.cfsb.instances.", func() {
					for _, node := range serviceCluster.Nodes {
						sp := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
						db, err := sp.Connect()
						So(err, ShouldBeNil)

						// TODO: Load the instance from the service cluster node database
						q := fmt.Sprintf(`SELECT instance_id FROM cfsb.instances WHERE instance_id = '%s'`, instanceID)
						err = db.Get(&s, q)
						So(err, ShouldBeNil)
						So(s, ShouldEqual, instanceID)
						db.Close()

						// and make sure that the following are true,
						//So(i.ClusterID, ShouldNotBeBlank)
						//So(i.ServiceID, ShouldEqual, ins.ServiceID)
						//So(i.PlanID, ShouldEqual, ins.PlanID)
						//So(i.OrganizationID, ShouldEqual, ins.OrganizationID)
						//So(i.SpaceID, ShouldEqual, ins.SpaceID)
					}
				})

				Convey("Each service cluster node for the instance's database should have bdr and btree_gist extension, and have the same count of bdr.bdr_nodes.", func() {
					var count int
					for _, node := range serviceCluster.Nodes {
						p := pg.NewPG(node.PG.IP, `7432`, `rdpg`, i.Database, i.Pass)
						db, err := p.Connect()
						So(err, ShouldBeNil)

						// extensions on all 5 nodes should have bdr, btree_gist
						exts := []string{"bdr", "btree_gist"}
						for _, ext := range exts {
							q := fmt.Sprintf(`SELECT extname FROM pg_extension WHERE extname ='%s'`, ext)
							err = db.Get(&s, q)
							So(err, ShouldBeNil)
							So(s, ShouldEqual, ext)
						}

						// replication group should have the same count as rdpg bdr.bdr_nodes;
						err = db.Get(&count, "SELECT count(node_status) FROM bdr.bdr_nodes WHERE node_status='r'")
						So(err, ShouldBeNil)
						So(count, ShouldEqual, len(serviceCluster.Nodes))
						db.Close()
					}
				})

				Convey("Bind", func() {
					bindingID := uuid.NewUUID().String()
					appGuid := uuid.NewUUID().String()

					type BindingBody struct {
						PlanID    string `json:"plan_id"`
						ServiceID string `json:"serive_is"`
						AppGuid   string `json:app_guid"`
					}

					bind := &BindingBody{
						PlanID:    planID,
						ServiceID: serviceID,
						AppGuid:   appGuid,
					}

					bindbody, err := json.Marshal(bind)
					So(err, ShouldBeNil)

					url := cfsbAPIURL("/v2/service_instances/" + instanceID + "/service_binding/" + bindingID)
					r, err := http.NewRequest("PUT", url, bytes.NewBuffer(bindbody))
					r.SetBasicAuth("cfadmin", "cfadmin")
					So(err, ShouldBeNil)

					httpClient := &http.Client{}
					resp, err := httpClient.Do(req)
					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusOK)

					// Binding that it returns should have values for it's fields

					decoder := json.NewDecoder(resp.Body)
					var b cfsb.Binding
					err = decoder.Decode(&b)
					So(err, ShouldBeNil)

					So(b.BindingID, ShouldEqual, bindingID)
					So(b.InstanceID, ShouldEqual, instanceID)
					dns := i.ExternalDNS()
					s_dns := strings.Split(dns, ":")
					So(b.Creds, ShouldNotBeBlank)
					creds := b.Creds
					So(creds.URI, ShouldEqual, i.URI())
					So(creds.DSN, ShouldEqual, i.DSN())
					So(creds.JDBCURI, ShouldEqual, "jdbc:"+i.URI())
					So(creds.Host, ShouldEqual, s_dns[0])
					So(creds.Port, ShouldEqual, s_dns[1])
					So(creds.UserName, ShouldEqual, i.User)
					So(creds.Password, ShouldEqual, i.Pass)
					So(creds.Database, ShouldEqual, i.Database)

					Convey("Each management cluster node should have a record of cfsb.binding and cfsb.credentials record for the bindingid.", func() {
						for _, node := range managementCluster.Nodes {
							p := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
							db, err := p.Connect()
							So(err, ShouldBeNil)

							q := fmt.Sprintf(`SELECT binding_id FROM cfsb.bindings WHERE binding_id = '%s'`, bindingID)
							err = db.Get(&s, q)
							So(err, ShouldBeNil)
							So(s, ShouldEqual, bindingID)

							q = fmt.Sprintf(`SELECT binding_id FROM cfsb.credentials WHERE binding_id = '%s'`, bindingID)
							err = db.Get(&s, q)
							So(err, ShouldBeNil)
							So(s, ShouldEqual, bindingID)

							db.Close()
						}
					})
					Convey("Un Bind", func() {
						url := cfsbAPIURL("/v2/service_instances/" + instanceID + "/service_binding/" + bindingID + "?service_id=" + serviceID + "&plan_id=" + planID)
						req, err = http.NewRequest("DELETE", url, nil)
						req.SetBasicAuth("cfadmin", "cfadmin")
						So(err, ShouldBeNil)

						httpClient := &http.Client{}
						resp, err := httpClient.Do(req)
						So(err, ShouldBeNil)
						So(resp.StatusCode, ShouldEqual, http.StatusOK)

						Convey("cfsb.binding and cfsb.credentials record for each management node should become inefective for the binding id .", func() {
							for _, node := range managementCluster.Nodes {
								p := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
								db, err := p.Connect()
								So(err, ShouldBeNil)

								q := fmt.Sprintf(`SELECT ineffective_at FROM cfsb.bindings WHERE binding_id = '%s'`, bindingID)
								err = db.Get(&s, q)
								So(err, ShouldBeNil)
								So(s, ShouldNotBeBlank)

								q = fmt.Sprintf(`SELECT ineffective_at FROM cfsb.credentials WHERE binding_id = '%s'`, bindingID)
								err = db.Get(&s, q)
								So(err, ShouldBeNil)
								So(s, ShouldNotBeBlank)

								db.Close()
							}
						})
					})
				})

				Convey("Deprovision", func() {
					url := cfsbAPIURL("/v2/service_instances/" + instanceID + "?service_id=" + serviceID + "&plan_id=" + planID)
					req, err = http.NewRequest("DELETE", url, nil)
					req.SetBasicAuth("cfadmin", "cfadmin")
					So(err, ShouldBeNil)

					httpClient := &http.Client{}
					resp, err := httpClient.Do(req)
					So(err, ShouldBeNil)
					So(resp.StatusCode, ShouldEqual, http.StatusOK)

					Convey("cfsb.instances for each management node should become inefective for the instanceID .", func() {
						for _, node := range managementCluster.Nodes {
							p := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
							db, err := p.Connect()
							So(err, ShouldBeNil)

							var ia pq.NullTime
							q := fmt.Sprintf(`SELECT ineffective_at FROM cfsb.instances WHERE instance_id='%s' LIMIT 1`, instanceID)
							err = db.Get(&ia, q)
							So(err, ShouldBeNil)
							So(ia.Valid, ShouldEqual, true)
							So(ia.Time, ShouldNotBeNil)
							db.Close()
						}
					})

					Convey("cfsb.instances for each service node should become inefective for the instanceID .", func() {
						for _, node := range serviceCluster.Nodes {
							p := pg.NewPG(node.PG.IP, `7432`, `rdpg`, `rdpg`, `admin`)
							db, err := p.Connect()
							So(err, ShouldBeNil)
							var ia pq.NullTime
							q := fmt.Sprintf(`SELECT ineffective_at FROM cfsb.instances WHERE instance_id='%s'`, instanceID)
							err = db.Get(&ia, q)
							So(err, ShouldBeNil)
							So(ia.Valid, ShouldEqual, true)
							So(ia.Time, ShouldNotBeNil)
							db.Close()
						}
					})

					Convey("Each node should NOT have the user and database anymore", func() {
						for _, node := range serviceCluster.Nodes {
							p := pg.NewPG(node.PG.IP, `7432`, `postgres`, `rdpg`, `admin`)
							db, err := p.Connect()
							So(err, ShouldBeNil)

							// user  should have been deleted on all service cluster nodes
							q := fmt.Sprintf(`SELECT rolname FROM pg_roles WHERE rolname='%s'`, i.User)
							err = db.Get(&s, q)
							So(err, ShouldBeNil)
							So(s, ShouldBeBlank)

							// database should be deleted all service cluster node
							q = fmt.Sprintf(`SELECT datname FROM pg_catalog.pg_database WHERE datname=?`, i.Database)
							err = db.Get(&s, q)
							So(err, ShouldBeNil)
							So(s, ShouldBeBlank)
							db.Close()
						}
					})
				})

			}) // Provision
		}) // catalog
	}) // integratetion workflow
}
