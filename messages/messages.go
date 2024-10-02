package messages

import "time"

type AddUserPermissionResponse struct{}
type AddUserPermissionRequest struct {
	Permission string
}
type ApproveUserPermissionResponse struct{}
type ApproveUserPermissionRequest struct {
	ApproverID string
	Permission string
}
type AwaitingApprovalResponse struct {
	Permissions []string
}
type CreateUserAccountResponse struct{}
type CreateUserAccountRequest struct {
	Permissions []string
}
type DeleteUserAccountResponse struct{}
type DeleteUserAccountRequest struct {
	DeletionRequestedAt time.Time
	DeletionScheduledAt time.Time
}
type GETUserResponse struct {
	AwaitingApproval   AwaitingApprovalResponse
	DeletionRequested  bool
	DeletionUndoWindow string
	Permissions        PermissionsGrantedResponse
	Username           string
}
type PermissionsGrantedResponse struct {
	Permissions []string
}
type SendNotificationsRequest struct {
	ApproverID     string
	PermissionType string
	RequesterID    string
}
type SendNotificationsResponse struct{}
type UndoDeleteUserAccountResponse struct{}
type UndoDeleteUserAccountRequest struct{}
type UserAccountOrchestrationInput struct {
	AwaitingApproval    []string
	Permissions         []string
	DeletionRequestedAt time.Time
}
type UserDetailsResponse struct {
	AwaitingApproval     AwaitingApprovalResponse
	DeletionRequested    bool
	DeletionRequestedAt  time.Time
	DeletionScheduledFor time.Time
	Permissions          PermissionsGrantedResponse
}
type VerifyApproverRequest struct {
	ApproverID string
	Permission string
}
type VerifyApproverResponse struct {
	Verified bool
}
