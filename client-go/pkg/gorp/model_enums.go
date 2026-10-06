package gorp

// LaunchMode represents the mode of a launch.
type LaunchMode string

type launchModeValuesType struct {
	Default LaunchMode
	Debug   LaunchMode
}

// LaunchModes contains the available launch mode values.
var LaunchModes = launchModeValuesType{
	Default: "DEFAULT",
	Debug:   "DEBUG",
}

// MergeType is the type of launch merge operation.
type MergeType string

type mergeTypeValuesType struct {
	Basic MergeType
	Deep  MergeType
}

// MergeTypes contains the available merge type values.
var MergeTypes = mergeTypeValuesType{
	Basic: "BASIC",
	Deep:  "DEEP",
}

// Status represents a test item or launch status.
type Status string

type statusValuesType struct {
	Passed      Status
	Failed      Status
	Stopped     Status
	Skipped     Status
	Interrupted Status
	Canceled    Status
	Info        Status
	Warn        Status
}

// Statuses contains the available status values.
var Statuses = statusValuesType{
	Passed:      "PASSED",
	Failed:      "FAILED",
	Stopped:     "STOPPED",
	Skipped:     "SKIPPED",
	Interrupted: "INTERRUPTED",
	Canceled:    "CANCELLED", //nolint:misspell // matches server-side spelling
	Info:        "INFO",
	Warn:        "WARN",
}

// TestItemType represents the type of a test item in the hierarchy.
type TestItemType string

type testItemTypeValuesType struct {
	Suite        TestItemType
	Story        TestItemType
	Test         TestItemType
	Scenario     TestItemType
	Step         TestItemType
	BeforeClass  TestItemType
	BeforeGroups TestItemType
	BeforeMethod TestItemType
	BeforeSuite  TestItemType
	BeforeTest   TestItemType
	AfterClass   TestItemType
	AfterGroups  TestItemType
	AfterMethod  TestItemType
	AfterSuite   TestItemType
	AfterTest    TestItemType
}

// TestItemTypes contains the available test item type values.
var TestItemTypes = testItemTypeValuesType{
	Suite:        "SUITE",
	Story:        "STORY",
	Test:         "TEST",
	Scenario:     "SCENARIO",
	Step:         "STEP",
	BeforeClass:  "BEFORE_CLASS",
	BeforeGroups: "BEFORE_GROUPS",
	BeforeMethod: "BEFORE_METHOD",
	BeforeSuite:  "BEFORE_SUITE",
	BeforeTest:   "BEFORE_TEST",
	AfterClass:   "AFTER_CLASS",
	AfterGroups:  "AFTER_GROUPS",
	AfterMethod:  "AFTER_METHOD",
	AfterSuite:   "AFTER_SUITE",
	AfterTest:    "AFTER_TEST",
}
