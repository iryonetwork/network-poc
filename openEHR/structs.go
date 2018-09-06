package openEHR

type Composer struct {
	ID   string `json:"/composer|identifier,omitempty"`
	Name string `json:"/composer|name,omitempty"`
}

type Repeating struct {
	Composer
	Language string `json:"/language,omitempty"`
}

type Shared struct {
	Repeating
	Timestamp string `json:"/context/start_time,omitempty"`
	Category  string `json:"/category,omitempty"`
}

type PersonalData struct {
	Shared
	BirthDate  string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/details[openEHR-DEMOGRAPHIC-ITEM_TREE.person_details.v1.0.0]/items[at0010],omitempty"`
	Gender     string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/details[openEHR-DEMOGRAPHIC-ITEM_TREE.person_details.v1.0.0]/items[at0017],omitempty"`
	FirstName  string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/identities[openEHR-DEMOGRAPHIC-PARTY_IDENTITY.person_name.v1]/details[at0001]/items[at0002],omitempty"`
	FamilyName string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/identities[openEHR-DEMOGRAPHIC-PARTY_IDENTITY.person_name.v1]/details[at0001]/items[at0003],omitempty"`
}

type Weight struct {
	Measure string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.body_weight.v2]/data[at0002]/events[at0003]:0/data[at0001]/items[at0004],omitempty"`
	Ts      string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.body_weight.v2]/data[at0002]/events[at0003]:0/time,omitempty"`
}

type Glucose struct {
	Measure string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.lab_test-blood_glucose.v1]/data[at0001]/events[at0002]:0/data[at0003]/items[at0078.2],omitempty"`
	Ts      string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.lab_test-blood_glucose.v1]/data[at0001]/events[at0002]:0/time,omitempty"`
}

type BloodPressure struct {
	Diastolyc string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.blood_pressure.v1]/data[at0001]/events[at0006]:0/data[at0003]/items[at0004],omitempty"`
	Systolic  string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.blood_pressure.v1]/data[at0001]/events[at0006]:0/data[at0003]/items[at0005],omitempty"`
	Ts        string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.blood_pressure.v1]/data[at0001]/events[at0006]:0/time,omitempty"`
}

type VitalSigns struct {
	Shared
	Weight
	Glucose
	BloodPressure
}

type All struct {
	VitalSigns
	PersonalData
}
