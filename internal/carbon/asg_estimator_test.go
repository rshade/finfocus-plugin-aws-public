package carbon

import (
	"math"
	"testing"
)

// TestASGEstimator_SingleInstance verifies that a single-instance ASG
// produces the same carbon as a direct EC2 estimator call.
func TestASGEstimator_SingleInstance(t *testing.T) {
	asgEst := NewASGEstimator()
	ec2Est := NewEstimator()

	ec2Carbon, ec2OK := ec2Est.EstimateCarbonGrams("m5.large", "us-east-1", 0.5, HoursPerMonth)
	asgCarbon, asgOK := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "m5.large",
		Region:          "us-east-1",
		DesiredCapacity: 1,
		Utilization:     0.5,
		Hours:           HoursPerMonth,
	})

	if !ec2OK || !asgOK {
		t.Fatal("both estimators should succeed for m5.large")
	}
	if math.Abs(asgCarbon-ec2Carbon) > 0.001 {
		t.Errorf("single-instance ASG carbon (%.2f) should match EC2 carbon (%.2f)", asgCarbon, ec2Carbon)
	}
}

// TestASGEstimator_MultiInstance verifies that carbon scales linearly with instance count.
func TestASGEstimator_MultiInstance(t *testing.T) {
	asgEst := NewASGEstimator()

	singleCarbon, ok := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "t3.micro",
		Region:          "us-east-1",
		DesiredCapacity: 1,
		Utilization:     0.5,
		Hours:           HoursPerMonth,
	})
	if !ok {
		t.Fatal("should succeed for t3.micro")
	}

	tripleCarbon, ok := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "t3.micro",
		Region:          "us-east-1",
		DesiredCapacity: 3,
		Utilization:     0.5,
		Hours:           HoursPerMonth,
	})
	if !ok {
		t.Fatal("should succeed for t3.micro × 3")
	}

	expected := singleCarbon * 3
	if math.Abs(tripleCarbon-expected) > 0.001 {
		t.Errorf("3-instance carbon (%.2f) should be 3× single (%.2f)", tripleCarbon, expected)
	}
}

// TestASGEstimator_UnknownInstanceType verifies that unknown instance types return false.
func TestASGEstimator_UnknownInstanceType(t *testing.T) {
	asgEst := NewASGEstimator()

	_, ok := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "x99.nonexistent",
		Region:          "us-east-1",
		DesiredCapacity: 1,
		Utilization:     0.5,
		Hours:           HoursPerMonth,
	})
	if ok {
		t.Error("should return false for unknown instance type")
	}
}

// TestASGEstimator_ZeroCapacity verifies that zero capacity returns 0 carbon.
func TestASGEstimator_ZeroCapacity(t *testing.T) {
	asgEst := NewASGEstimator()

	carbonGrams, ok := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "m5.large",
		Region:          "us-east-1",
		DesiredCapacity: 0,
		Utilization:     0.5,
		Hours:           HoursPerMonth,
	})
	if !ok {
		t.Fatal("zero capacity should succeed")
	}
	if carbonGrams != 0 {
		t.Errorf("zero capacity should produce 0 carbon, got %.2f", carbonGrams)
	}
}

// TestASGEstimator_UtilizationPassthrough verifies that utilization is passed to the EC2 estimator.
func TestASGEstimator_UtilizationPassthrough(t *testing.T) {
	asgEst := NewASGEstimator()

	lowUtil, ok1 := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "m5.large",
		Region:          "us-east-1",
		DesiredCapacity: 1,
		Utilization:     0.1,
		Hours:           HoursPerMonth,
	})
	highUtil, ok2 := asgEst.EstimateCarbonGrams(ASGConfig{
		InstanceType:    "m5.large",
		Region:          "us-east-1",
		DesiredCapacity: 1,
		Utilization:     0.9,
		Hours:           HoursPerMonth,
	})

	if !ok1 || !ok2 {
		t.Fatal("both utilization levels should succeed")
	}
	if highUtil <= lowUtil {
		t.Errorf("higher utilization (%.2f) should produce more carbon than lower (%.2f)", highUtil, lowUtil)
	}
}
