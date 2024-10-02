package orchestrations

import (
	"errors"
	"fmt"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	msgs "github.com/temporal-sa/temporal-entity-lifecycle-go/messages"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/user_account_state"
	wf "go.temporal.io/sdk/workflow"
)

type UserAccountOrchestrationHandler struct{}

func New() (*UserAccountOrchestrationHandler, error) {
	return &UserAccountOrchestrationHandler{}, nil
}

func (h *UserAccountOrchestrationHandler) Orchestration(ctx wf.Context, in msgs.UserAccountOrchestrationInput) error {
	state, err := user_account_state.New(ctx, user_account_state.WithSnapshot(in))
	if err != nil {
		return errors.Join(errors.New("unable to initialize user_account_state"), err)
	}
	err = wf.SetUpdateHandler(ctx, constants.CreateUserAccountUpdateHandlerName,
		func(inner wf.Context, req msgs.CreateUserAccountRequest) (msgs.CreateUserAccountResponse, error) {
			return msgs.CreateUserAccountResponse{}, state.CreateUser(req)
		})
	if err != nil {
		return errors.Join(errors.New(fmt.Sprintf("unable to set %s UpdateHandler",
			constants.CreateUserAccountUpdateHandlerName)), err)
	}
	err = wf.SetUpdateHandler(ctx, constants.AddUserPermissionUpdateHandlerName,
		func(inner wf.Context, req msgs.AddUserPermissionRequest) (msgs.AddUserPermissionResponse, error) {
			return msgs.AddUserPermissionResponse{}, state.RequestAddPermission(req)
		})
	if err != nil {
		return errors.Join(errors.New(
			fmt.Sprintf("unable to set %s UpdateHandler", constants.AddUserPermissionUpdateHandlerName)), err)
	}
	err = wf.SetUpdateHandler(ctx, constants.ApproveUserPermissionUpdateHandlerName,
		func(inner wf.Context, req msgs.ApproveUserPermissionRequest) (msgs.ApproveUserPermissionResponse, error) {
			return msgs.ApproveUserPermissionResponse{}, state.RequestApprovePermission(inner, req)
		})
	if err != nil {
		return errors.Join(
			errors.New(
				fmt.Sprintf(
					"unable to set %s UpdateHandler", constants.ApproveUserPermissionUpdateHandlerName)),
			err)
	}
	err = wf.SetUpdateHandler(ctx, constants.DeleteUserAccountUpdateHandlerName,
		func(inner wf.Context, req msgs.DeleteUserAccountRequest) (msgs.DeleteUserAccountResponse, error) {
			state.RequestDeletion(req)
			return msgs.DeleteUserAccountResponse{}, nil
		})
	if err != nil {
		return errors.Join(errors.New(
			fmt.Sprintf("unable to set %s UpdateHandler", constants.DeleteUserAccountUpdateHandlerName)), err)
	}
	err = wf.SetUpdateHandler(ctx, constants.UndoDeleteUserAccountUpdateHandlerName,
		func(inner wf.Context, req msgs.UndoDeleteUserAccountRequest) (msgs.UndoDeleteUserAccountResponse, error) {
			requestUndoDeletionErr := state.RequestUndoDeletion(req)
			return msgs.UndoDeleteUserAccountResponse{}, requestUndoDeletionErr
		})
	if err != nil {
		return errors.Join(errors.New(fmt.Sprintf("unable to set %s UpdateHandler",
			constants.UndoDeleteUserAccountUpdateHandlerName)), err)
	}
	err = wf.SetQueryHandler(ctx, constants.AwaitingApprovalQueryHandlerName,
		func() (msgs.AwaitingApprovalResponse, error) {
			return state.AwaitingApproval(), nil
		})
	if err != nil {
		return errors.Join(errors.New(
			fmt.Sprintf("unable to set %s QueryHandler", constants.AwaitingApprovalQueryHandlerName)), err)
	}
	err = wf.SetQueryHandler(ctx, constants.PermissionsGrantedQueryHandlerName,
		func() (msgs.PermissionsGrantedResponse, error) {
			return state.Permissions(), nil
		})
	if err != nil {
		return errors.Join(errors.New(
			fmt.Sprintf("unable to set %s QueryHandler", constants.PermissionsGrantedQueryHandlerName)), err)
	}
	err = wf.SetQueryHandler(ctx, constants.UserDetailsQueryHandlerName,
		func() (msgs.UserDetailsResponse, error) {
			return state.UserDetails(), nil
		})
	if err != nil {
		return errors.Join(errors.New(
			fmt.Sprintf("unable to set %s QueryHandler", constants.UserDetailsQueryHandlerName)), err)
	}
	err = wf.Await(ctx, func() bool { return state.Deleted() || wf.GetInfo(ctx).GetContinueAsNewSuggested() })
	if err != nil {
		return errors.Join(errors.New("wait cancelled"), err)
	}
	if wf.GetInfo(ctx).GetContinueAsNewSuggested() {
		return wf.NewContinueAsNewError(ctx, h.Orchestration, msgs.UserAccountOrchestrationInput{
			AwaitingApproval:    state.AwaitingApproval().Permissions,
			DeletionRequestedAt: state.DeletionRequestedAt(),
			Permissions:         state.Permissions().Permissions,
		})
	}
	return nil
}
