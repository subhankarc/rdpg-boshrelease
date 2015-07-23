package rdpg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/starkandwayne/rdpgd/cfsb"
	"github.com/starkandwayne/rdpgd/instances"
	"github.com/starkandwayne/rdpgd/pg"
	"github.com/starkandwayne/rdpgd/uuid"
)

func init() {
	os.Setenv("RDPGD_PB_PORT", "7432")
}

func cfsbAPIURL(path string) string {
	return fmt.Sprintf(`http://cfadmin:cfadmin@10.244.2.2:8888%s`, path)
}

func TestIntegration(t *testing.T) {
	Convey(`RDPG System Integration, given two service clusters`, t, func() {

		Convey(`When Pool Size + 1 databases are assigned`, func() {
			// TODO: Provision Pool Size + 1 databases here...
			pSize := os.Getenv(`RDPGD_POOL_SIZE`)
			poolSize, err := strconv.Atoi(pSize)
			So(err, ShouldBeNil)
			So(poolSize, ShouldEqual, 10)

			organizationID := uuid.NewUUID().String()
			spaceID := uuid.NewUUID().String()

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

			serviceID := c.Services[0].ServiceID
			planID := c.Services[0].Plans[0].PlanID

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

			sc1Count := 0
			sc2Count := 0

			for ps := 0; ps < poolSize; ps++ {
				instanceID := uuid.NewUUID().String()
				insbody, err := json.Marshal(ins)
				So(err, ShouldBeNil)
				url := cfsbAPIURL("/v2/service_instances/" + instanceID)
				req, err := http.NewRequest("PUT", url, bytes.NewBuffer(insbody))
				req.SetBasicAuth("cfadmin", "cfadmin")

				httpClient := &http.Client{}
				resp, err := httpClient.Do(req)
				So(err, ShouldBeNil)
				So(resp.StatusCode, ShouldEqual, http.StatusOK)

				p := pg.NewPG(`10.244.2.2`, `7432`, `rdpg`, `rdpg`, `admin`)
				db, err := p.Connect()
				i := instances.Instance{}
				sq := fmt.Sprintf(`SELECT id, cluster_id,instance_id, service_id, plan_id, organization_id, space_id, dbname, dbuser, dbpass FROM cfsb.instances WHERE instance_id=lower('%s') LIMIT 1`, instanceID)
				err = db.Get(&i, sq)
				if i.ClusterID == "rdpgsc1" {
					sc1Count += 1
				}
				if i.ClusterID == "rdpgsc2" {
					sc2Count += 1
				}

			}
			Convey(`Databases should be assigned to more than one service cluster.`, func() {
				// TODO: count the # assigned on each service cluster and
				// Both should be > 0
				So(sc1Count, ShouldBeGreaterThan, 0)
				So(sc2Count, ShouldBeGreaterThan, 0)
			})
		})
	})
}
