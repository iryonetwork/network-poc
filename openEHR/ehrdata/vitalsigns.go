package ehrdata

import (
	"fmt"

	"github.com/iryonetwork/network-poc/openEHR"
	"github.com/iryonetwork/network-poc/state"
)

func NewVitalSigns(state *state.State) *openEHR.VitalSigns {
	return &openEHR.VitalSigns{
		Shared: openEHR.Shared{
			Repeating: state.PersonalData.Repeating,
			Category:  "433|event",
			Timestamp: timestamp(),
		},
	}
}

func AddVitalSigns(d *openEHR.VitalSigns, weight, glucose, bpSystolic, bpDiastolic string) error {
	if weight == "" && glucose == "" && bpDiastolic == "" && bpSystolic == "" {
		return fmt.Errorf("Please insert data")
	}

	if weight != "" {
		d.Weight = openEHR.Weight{Ts: timestamp(), Measure: addUnit(weight, "kg")}
	}

	if glucose != "" {
		d.Glucose = openEHR.Glucose{Ts: timestamp(), Measure: addUnit(glucose, "mmol/l")}
	}

	if bpSystolic != "" || bpDiastolic != "" {
		d.BloodPressure = openEHR.BloodPressure{Ts: timestamp(), Systolic: addUnit(bpSystolic, "mm[Hg]"), Diastolic: addUnit(bpDiastolic, "mm[Hg]")}
	}

	return nil
}
