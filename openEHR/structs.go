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

type PersonalDataFields struct {
	BirthDate  string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/details[openEHR-DEMOGRAPHIC-ITEM_TREE.person_details.v1.0.0]/items[at0010],omitempty"`
	Gender     string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/details[openEHR-DEMOGRAPHIC-ITEM_TREE.person_details.v1.0.0]/items[at0017],omitempty"`
	FirstName  string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/identities[openEHR-DEMOGRAPHIC-PARTY_IDENTITY.person_name.v1]/details[at0001]/items[at0002],omitempty"`
	FamilyName string `json:"/content[openEHR-DEMOGRAPHIC-PERSON.person.v1]/identities[openEHR-DEMOGRAPHIC-PARTY_IDENTITY.person_name.v1]/details[at0001]/items[at0003],omitempty"`
}

type PersonalData struct {
	Shared
	PersonalDataFields
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
	Diastolic string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.blood_pressure.v1]/data[at0001]/events[at0006]:0/data[at0003]/items[at0004],omitempty"`
	Systolic  string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.blood_pressure.v1]/data[at0001]/events[at0006]:0/data[at0003]/items[at0005],omitempty"`
	Ts        string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.blood_pressure.v1]/data[at0001]/events[at0006]:0/time,omitempty"`
}

type HeartRate struct {
	Name  string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.pulse.v1]/data[at0002]/events[at0003]:0/data[at0001]/items[at0004]|name,omitempty"`
	Value string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.pulse.v1]/data[at0002]/events[at0003]:0/data[at0001]/items[at0004],omitempty"`
	Ts    string `json:"/content[openEHR-EHR-COMPOSITION.encounter.v1]/context/other_context/items[openEHR-EHR-OBSERVATION.pulse.v1]/data[at0002]/events[at0003]:0/time,omitempty"`
}

type ECG struct {
	Shared
	RRRate                  string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0013]/value,omitempty"`
	PRInterval              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0012]/value,omitempty"`
	QRSDuration             string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0014]/value,omitempty"`
	QTInterval              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0007]/value,omitempty"`
	QTCInterval             string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0008]/value,omitempty"`
	PAxis                   string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0023]/items[at0020]/value,omitempty"`
	QRSAxis                 string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0023]/items[at0021]/value,omitempty"`
	TAxis                   string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0023]/items[at0022]/value,omitempty"`
	PAmplitude              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0029]/items[at0041]/value,omitempty"`
	SAmplitude              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0035]/items[at0053]/value,omitempty"`
	RAmplitude              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0039]/items[at0050]/value,omitempty"`
	AutomaticInterpretation string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0009]/value,omitempty"`
	DeviceName              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/protocol[at0003]/items[at0076|openEHR-EHR-CLUSTER.device.v1]/items[at0001]/value,omitempty"`
	DeviceType              string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/protocol[at0003]/items[at0076|openEHR-EHR-CLUSTER.device.v1]/items[at0003]/value,omitempty"`
	DeviceManufacturer      string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/protocol[at0003]/items[at0076|openEHR-EHR-CLUSTER.device.v1]/items[at0004]/value,omitempty"`
	AttachmentName          string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0083|openEHR-EHR-CLUSTER.multimedia.v0]/items[at0002]/value,omitempty"`
	AttachmentType          string  `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0083|openEHR-EHR-CLUSTER.multimedia.v0]/items[at0001]/value|media_type,omitempty"`
	AttachmentData          *string `json:"/content[openEHR-EHR-OBSERVATION.ecg_result.v0]/data[at0001]/events[at0002]/data[at0005]/items[at0083|openEHR-EHR-CLUSTER.multimedia.v0]/items[at0001]/value|data,omitempty"`
}

type VitalSignsFields struct {
	Weight
	Glucose
	BloodPressure
}

type VitalSigns struct {
	Shared
	VitalSignsFields
}

type All struct {
	Shared
	PersonalDataFields
	VitalSignsFields
}
