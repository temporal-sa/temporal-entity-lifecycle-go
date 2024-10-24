package orchestrations

import (
	"errors"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/messages"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/orchestrations/activity_handler"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/worker"
	"testing"
	"time"
)

type UnitTestSuite struct {
	suite.Suite
	testsuite.WorkflowTestSuite

	env *testsuite.TestWorkflowEnvironment
}

func (s *UnitTestSuite) AfterTest(suiteName, testName string) {
	s.env.AssertExpectations(s.T())
}

func (s *UnitTestSuite) SetupTest() {
	s.env = s.NewTestWorkflowEnvironment()
}

func TestUnitTestSuite(t *testing.T) {
	suite.Run(t, new(UnitTestSuite))
}

func (s *UnitTestSuite) Test_Orchestration_HandleAddPermission() {
	h, err := New()
	s.Nil(err)
	s.env.SetTestTimeout(time.Second * 5)
	uc := &updateCallbacks{t: s.T()}
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.AddUserPermissionUpdateHandlerName, "1", uc,
			messages.AddUserPermissionRequest{
				Permission: constants.PermissionTypeReadFiles,
			})
	}, time.Second*1)
	s.env.ExecuteWorkflow(h.Orchestration, messages.UserAccountOrchestrationInput{})
	s.True(s.env.IsWorkflowCompleted())
	v, err := s.env.QueryWorkflow(constants.AwaitingApprovalQueryHandlerName)
	s.Nil(err)
	awaitingApproval := messages.AwaitingApprovalResponse{}
	err = v.Get(&awaitingApproval)
	s.Nil(err)
	expected := messages.AwaitingApprovalResponse{
		Permissions: []string{constants.PermissionTypeReadFiles},
	}
	s.Equal(expected, awaitingApproval)
}

func (s *UnitTestSuite) Test_Orchestration_HandleApprovePermission() {
	h, err := New()
	s.Nil(err)
	s.env.SetTestTimeout(time.Second * 5)
	uc := &updateCallbacks{t: s.T()}
	permissionsKey := temporal.NewSearchAttributeKeyKeywordList(constants.PermissionsSearchAttributeKey)
	permissionsSearchAttrState1 := temporal.NewSearchAttributes(permissionsKey.ValueSet([]string{}))
	s.env.OnUpsertTypedSearchAttributes(permissionsSearchAttrState1).Return(nil).Once()
	permissionsSearchAttrState2 := temporal.NewSearchAttributes(permissionsKey.ValueSet([]string{constants.PermissionTypeReadFiles}))
	s.env.OnUpsertTypedSearchAttributes(permissionsSearchAttrState2).Return(nil).Once()
	approvalsKey := temporal.NewSearchAttributeKeyKeywordList(constants.AwaitingApprovalSearchAttributeKey)
	approvalsSearchAttrState1 := temporal.NewSearchAttributes(approvalsKey.ValueSet([]string{constants.PermissionTypeReadFiles}))
	s.env.OnUpsertTypedSearchAttributes(approvalsSearchAttrState1).Return(nil).Once()
	approvalsSearchAttrState2 := temporal.NewSearchAttributes(approvalsKey.ValueSet([]string{}))
	s.env.OnUpsertTypedSearchAttributes(approvalsSearchAttrState2).Return(nil).Once()
	s.env.RegisterActivityWithOptions(new(activity_handler.Handler).VerifyApprover, activity.RegisterOptions{
		Name: "VerifyApprover",
	})
	s.env.OnActivity(new(activity_handler.Handler).VerifyApprover, mock.Anything, messages.VerifyApproverRequest{
		ApproverID: "bobsaget@temporal.io",
		Permission: constants.PermissionTypeReadFiles,
	}).Return(messages.VerifyApproverResponse{Verified: true}, nil)
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.AddUserPermissionUpdateHandlerName, "1", uc,
			messages.AddUserPermissionRequest{
				Permission: constants.PermissionTypeReadFiles,
			})
	}, time.Second*1)
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.ApproveUserPermissionUpdateHandlerName, "1", uc,
			messages.ApproveUserPermissionRequest{
				Permission: constants.PermissionTypeReadFiles,
				ApproverID: "bobsaget@temporal.io",
			})
	}, time.Second*2)
	s.env.ExecuteWorkflow(h.Orchestration, messages.UserAccountOrchestrationInput{})
	s.True(s.env.IsWorkflowCompleted())
	v, err := s.env.QueryWorkflow(constants.PermissionsGrantedQueryHandlerName)
	s.Nil(err)
	granted := messages.PermissionsGrantedResponse{}
	err = v.Get(&granted)
	s.Nil(err)
	expected := messages.PermissionsGrantedResponse{
		Permissions: []string{constants.PermissionTypeReadFiles},
	}
	s.Equal(expected, granted)
}

func (s *UnitTestSuite) Test_Orchestration_HandleApprovePermission_UnauthorizedApprover() {
	h, err := New()
	s.Nil(err)
	s.env.SetTestTimeout(time.Second * 5)
	uc := &updateCallbacks{t: s.T()}
	s.env.RegisterActivityWithOptions(new(activity_handler.Handler).VerifyApprover, activity.RegisterOptions{
		Name: "VerifyApprover",
	})
	s.env.OnActivity(new(activity_handler.Handler).VerifyApprover, mock.Anything, messages.VerifyApproverRequest{
		ApproverID: "bobsaget@temporal.io",
		Permission: constants.PermissionTypeReadFiles,
	}).Return(messages.VerifyApproverResponse{Verified: false}, nil)
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.AddUserPermissionUpdateHandlerName, "1", uc,
			messages.AddUserPermissionRequest{
				Permission: constants.PermissionTypeReadFiles,
			})
	}, time.Second*1)
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.ApproveUserPermissionUpdateHandlerName, "1", uc,
			messages.ApproveUserPermissionRequest{
				Permission: constants.PermissionTypeReadFiles,
				ApproverID: "bobsaget@temporal.io",
			})
	}, time.Second*2)
	s.env.ExecuteWorkflow(h.Orchestration, messages.UserAccountOrchestrationInput{})
	s.True(s.env.IsWorkflowCompleted())
	s.Error(uc.Error())
	s.Equal("bobsaget@temporal.io cannot grant permission read_files", uc.Error().Error())
}

func (s *UnitTestSuite) Test_Orchestration_HandleCreateUpdate() {
	h, err := New()
	s.Nil(err)
	s.env.SetTestTimeout(time.Second * 5)
	uc := &updateCallbacks{t: s.T()}
	attributeKey := temporal.NewSearchAttributeKeyKeywordList(constants.PermissionsSearchAttributeKey)
	attributes := temporal.NewSearchAttributes(attributeKey.ValueSet([]string{constants.PermissionTypeGrantPermissions}))
	s.env.OnUpsertTypedSearchAttributes(attributes).Return(nil).Once()
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.CreateUserAccountUpdateHandlerName, "1", uc,
			messages.CreateUserAccountRequest{
				Permissions: []string{constants.PermissionTypeGrantPermissions},
			})
	}, time.Second*1)
	s.env.ExecuteWorkflow(h.Orchestration, messages.UserAccountOrchestrationInput{})
	s.True(s.env.IsWorkflowCompleted())
	v, err := s.env.QueryWorkflow(constants.PermissionsGrantedQueryHandlerName)
	s.Nil(err)
	granted := messages.PermissionsGrantedResponse{}
	err = v.Get(&granted)
	s.Nil(err)
	expected := messages.PermissionsGrantedResponse{
		Permissions: []string{constants.PermissionTypeGrantPermissions},
	}
	s.Equal(expected, granted)
}

func (s *UnitTestSuite) Test_Orchestration_HandleDeleteUpdate() {
	h, err := New()
	s.Nil(err)
	s.env.SetTestTimeout(time.Second * 5)
	uc := &updateCallbacks{t: s.T()}
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.DeleteUserAccountUpdateHandlerName, "1", uc,
			messages.DeleteUserAccountRequest{})
	}, time.Second*1)
	s.env.ExecuteWorkflow(h.Orchestration, messages.UserAccountOrchestrationInput{})
	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Nil(err)
}

func (s *UnitTestSuite) Test_Orchestration_HandleUndoDeleteUpdate() {
	// In the event that deletion is undone within the soft-delete time window the workflow should revert to it's normal
	// behavior: unending execution. In tests this looks like a timeout since the workflow DOES NOT complete within the
	// test timeout window.
	h, err := New()
	s.Nil(err)
	s.env.SetTestTimeout(time.Second * 5)
	uc := &updateCallbacks{t: s.T()}
	s.env.RegisterDelayedCallback(func() {
		s.env.UpdateWorkflow(constants.DeleteUserAccountUpdateHandlerName, "1", uc,
			messages.DeleteUserAccountRequest{})
	}, time.Second*1)
	s.env.RegisterDelayedCallback(func() {
		// Want to verify that this test works? Comment out this callback and the test will fail.
		s.env.UpdateWorkflow(constants.UndoDeleteUserAccountUpdateHandlerName, "1", uc,
			messages.UndoDeleteUserAccountRequest{})
	}, time.Second*2)
	s.env.ExecuteWorkflow(h.Orchestration, messages.UserAccountOrchestrationInput{})
	s.True(s.env.IsWorkflowCompleted())
	err = s.env.GetWorkflowError()
	s.Error(err)
	s.IsTypef(&temporal.TimeoutError{}, errors.Unwrap(err), "")
}

func (s *UnitTestSuite) Test_Orchestration_ReplayHistory() {
	oh, err := New()
	s.Nil(err)
	r := worker.NewWorkflowReplayer()
	r.RegisterWorkflow(oh.Orchestration)
	err = r.ReplayWorkflowHistoryFromJSONFile(s.GetLogger(), "../fixtures/event_history.json")
	s.Nil(err)
}

// updateCallbacks are necessary for testing updates AND are an excellent affordance for debugging
type updateCallbacks struct {
	t   *testing.T
	err error
}

func (uc *updateCallbacks) Accept() {
	// uc.t.Log("ACCEPT")
}

func (uc *updateCallbacks) Reject(err error) {
	uc.err = err
	// uc.t.Logf("rejected err—%s", err.Error())
}

func (uc *updateCallbacks) Complete(success interface{}, err error) {
	if err != nil {
		uc.err = err
	}
	if err != nil {
		// uc.t.Logf("complete err—%s", err.Error())
		return
	}
	// uc.t.Logf("complete success %v", success)
}

func (uc *updateCallbacks) Error() error {
	return uc.err
}
