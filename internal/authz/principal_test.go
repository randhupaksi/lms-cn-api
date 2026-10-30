package authz

import "testing"

func TestCurriculumCapabilities(t *testing.T) {
	curriculum := Principal{Role: "curriculum"}
	if !curriculum.IsTeachingStaff() || !curriculum.CanCoordinateAcademics() {
		t.Fatal("curriculum must combine assigned-course teaching with academic coordination")
	}
	if curriculum.IsSchoolAdmin() {
		t.Fatal("curriculum must not inherit administrative security scope")
	}
}
