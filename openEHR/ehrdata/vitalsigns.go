package ehrdata

import (
	"github.com/iryonetwork/network-poc/config"
	"github.com/iryonetwork/network-poc/openEHR"
)

func NewVitalSigns(config *config.Config) *openEHR.VitalSigns {
	return &openEHR.VitalSigns{
		Shared: config.PersonalData.Shared,
	}
}

func AddVitalSigns(d *openEHR.VitalSigns, weight, glucose, bpSystolic, bpDiastolic string) {
	if weight != "" {
		d.Weight = openEHR.Weight{Ts: timestamp(), Measure: weight}
	}

	if glucose != "" {
		d.Glucose = openEHR.Glucose{Ts: timestamp(), Measure: glucose}
	}

	if bpSystolic != "" || bpDiastolic != "" {
		d.BloodPressure = openEHR.BloodPressure{Ts: timestamp(), Systolic: bpSystolic, Diastolyc: bpDiastolic}
	}
}
