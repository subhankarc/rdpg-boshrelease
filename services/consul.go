package services

import (
	"errors"
	"fmt"

	"github.com/starkandwayne/rdpgd/log"
)

func (s *Service) ConfigureConsul() (err error) {
	log.Trace(fmt.Sprintf("services#Service.ConfigureConsul()..."))
	// TODO: Adjust for cluster role...

	return errors.New(`services.Service#Configure("consul") is not yet implemented`)
}
