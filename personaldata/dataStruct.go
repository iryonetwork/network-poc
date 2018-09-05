package personaldata

import "github.com/iryonetwork/network-poc/config"

type Data struct {
	config     *config.Config
	Category   string `json:"/category"`
	ID         string `json:"/composer|identifier"`
	Name       string `json:"/composer|name"`
	Timstamp   string `json:"/context/start_time"`
	Language   string `json:"/language"`
	BirthDate  string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/details[openEHR-DEMOGRAPHIC-ITEM_TREE.person_details.v1.0.0]/items[at0010]"`
	Gender     string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/details[openEHR-DEMOGRAPHIC-ITEM_TREE.person_details.v1.0.0]/items[at0017]"`
	FirstName  string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/identities[openEHR-DEMOGRAPHIC-PARTY_IDENTITY.person_name.v1]/details[at0001]/items[at0002]"`
	FamilyName string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/identities[openEHR-DEMOGRAPHIC-PARTY_IDENTITY.person_name.v1]/details[at0001]/items[at0003]"`
}
