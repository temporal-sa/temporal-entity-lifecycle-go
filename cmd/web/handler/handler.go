package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/constants"
	"github.com/temporal-sa/temporal-entity-lifecycle-go/messages"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/api/workflowservice/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/zap"
	"net/http"
	"strings"
	"time"
)

type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type Handler struct {
	c  client.Client
	l  Logger
	ns string
}

type Option func(*Handler)

func New(c client.Client, namespace string, opts ...Option) (*Handler, error) {
	logger, _ := zap.NewProduction()
	h := &Handler{
		c:  c,
		ns: namespace,
		l:  logger,
	}
	if h.c == nil {
		return nil, errors.New("temporal client required & missing")
	}
	for _, o := range opts {
		o(h)
	}
	return h, nil
}

func (h Handler) GETApprovePermission(gc *gin.Context) {
	gc.HTML(http.StatusOK, "approve_permission.html", nil)
}

func (h Handler) GETCreateUser(gc *gin.Context) {
	gc.HTML(http.StatusOK, "create_user.html", nil)
}

func (h Handler) GETRequestPermission(gc *gin.Context) {
	listWorkflowReq := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: h.ns,
		Query:     "`ExecutionStatus`=\"Running\"",
	}
	listResp, err := h.c.ListWorkflow(gc.Request.Context(), listWorkflowReq)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	users := make([]string, 0)
	for _, e := range listResp.GetExecutions() {
		users = append(users, e.GetExecution().GetWorkflowId())
	}
	gc.HTML(http.StatusOK, "request_permission.html", users)
}

func (h Handler) GETUser(gc *gin.Context) {
	if gc.Query("id") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "id required and missing")
		return
	}
	ev, err := h.c.QueryWorkflow(gc.Request.Context(), gc.Query("id"), "",
		constants.UserDetailsQueryHandlerName)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	ud := messages.UserDetailsResponse{}
	err = ev.Get(&ud)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	gc.HTML(http.StatusOK, "user.html", messages.GETUserResponse{
		AwaitingApproval:   ud.AwaitingApproval,
		DeletionRequested:  ud.DeletionRequested,
		DeletionUndoWindow: ud.DeletionScheduledFor.Sub(time.Now().UTC()).String(),
		Permissions:        ud.Permissions,
		Username:           gc.Query("id"),
	})
}

func (h Handler) GETUsers(gc *gin.Context) {
	// In a production app we ought to paginate through the executions. Since this is a demo we can assume that the
	// number of open executions will fit in a single request.
	listWorkflowReq := &workflowservice.ListWorkflowExecutionsRequest{
		Namespace: h.ns,
		Query:     "`ExecutionStatus`=\"Running\"",
	}
	if gc.Query("permission") != "" {
		queryString := "`ExecutionStatus`=\"Running\" AND `permissions`=\"{NAME}\""
		queryString = strings.Replace(queryString, "{NAME}", gc.Query("permission"), 1)
		listWorkflowReq.Query = queryString
	}
	var listResponse *workflowservice.ListWorkflowExecutionsResponse
	for listResponse == nil {
		tempListResponse, err := h.c.ListWorkflow(gc.Request.Context(), listWorkflowReq)
		if err != nil {
			_ = gc.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		if gc.Query("flashUserCreated") != "" {
			// Poll ListWorkflow until we find the created user: this is for demonstration purposes only and NOT
			// indicative of best practices
			for _, e := range tempListResponse.GetExecutions() {
				if e.GetExecution().GetWorkflowId() == gc.Query("flashUserCreated") {
					listResponse = tempListResponse
					break
				}
				time.Sleep(100 * time.Millisecond)
			}
		} else if gc.Query("permissionRequested") != "" && gc.Query("username") != "" {
			// Poll ListWorkflow until we find the requested permission: this is for demonstration purposes only
			// and NOT indicative of best practices
			for _, e := range tempListResponse.GetExecutions() {
				if e.GetExecution().GetWorkflowId() == gc.Query("username") {
					if e.GetSearchAttributes().GetIndexedFields()[constants.AwaitingApprovalSearchAttributeKey] != nil {
						awaitingApprovals := make([]string, 0)
						awaitingApprovalsBytes := e.GetSearchAttributes().
							GetIndexedFields()[constants.AwaitingApprovalSearchAttributeKey].GetData()
						if len(awaitingApprovalsBytes) > 0 {
							err := json.Unmarshal(awaitingApprovalsBytes, &awaitingApprovals)
							if err != nil {
								_ = gc.AbortWithError(http.StatusInternalServerError, err)
								return
							}
						}
						for _, aa := range awaitingApprovals {
							if aa == gc.Query("permissionRequested") {
								listResponse = tempListResponse
								break
							}
						}
					}
				}
			}
		} else {
			listResponse = tempListResponse
		}
	}
	type User struct {
		Username          string
		AwaitingApprovals []string
	}
	type UsersResponse struct {
		AdminUsername                  string
		Users                          []User
		FlashUserCreatedMessage        string
		FlashUserAlreadyCreatedMessage string
	}
	response := UsersResponse{
		Users: make([]User, 0),
	}
	if gc.Query("flashUserCreated") != "" {
		response.FlashUserCreatedMessage = "Created user " + gc.Query("flashUserCreated")
	}
	if gc.Query("flashUserAlreadyCreated") != "" {
		msg := fmt.Sprintf("User %s has already been created", gc.Query("flashUserAlreadyCreated"))
		response.FlashUserAlreadyCreatedMessage = msg
	}
	for _, e := range listResponse.GetExecutions() {
		awaitingApprovals := make([]string, 0)
		awaitingApprovalsBytes := e.GetSearchAttributes().
			GetIndexedFields()[constants.AwaitingApprovalSearchAttributeKey].GetData()
		if len(awaitingApprovalsBytes) > 0 {
			err := json.Unmarshal(awaitingApprovalsBytes, &awaitingApprovals)
			if err != nil {
				_ = gc.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}
		permissions := make([]string, 0)
		permissionsBytes := e.GetSearchAttributes().GetIndexedFields()[constants.PermissionsSearchAttributeKey].
			GetData()
		if len(permissionsBytes) > 0 {
			err := json.Unmarshal(permissionsBytes, &permissions)
			if err != nil {
				_ = gc.AbortWithError(http.StatusInternalServerError, err)
				return
			}
		}
		for _, p := range permissions {
			if p == constants.PermissionTypeGrantPermissions && response.AdminUsername == "" {
				response.AdminUsername = e.GetExecution().GetWorkflowId()
				break
			}
		}
		u := User{
			Username: e.GetExecution().GetWorkflowId(),
		}
		if len(awaitingApprovals) > 0 {
			u.AwaitingApprovals = awaitingApprovals
		} else {
			u.AwaitingApprovals = []string{}
		}
		response.Users = append(response.Users, u)
	}
	gc.HTML(http.StatusOK, "users.html", response)
}

func (h Handler) POSTApprovePermission(gc *gin.Context) {
	if gc.PostForm("requester_username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "requester_username required and missing")
		return
	}
	if gc.PostForm("approver_username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "approver_username required and missing")
		return
	}
	if gc.PostForm("permission_type") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "permission_type required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("requester_username"),
		UpdateName: constants.ApproveUserPermissionUpdateHandlerName,
		Args: []interface{}{
			&messages.ApproveUserPermissionRequest{
				ApproverID: gc.PostForm("approver_username"),
				Permission: gc.PostForm("permission_type"),
			},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := messages.ApproveUserPermissionResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("requester_username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}

func (h Handler) POSTCreateUser(gc *gin.Context) {
	if gc.Request.FormValue("username") == "" {
		gc.String(http.StatusBadRequest, "username required and missing")
		return
	}
	workflowID := gc.Request.FormValue("username")
	opts := client.StartWorkflowOptions{
		ID:                                       workflowID,
		TaskQueue:                                constants.EntityTaskQueueName,
		WorkflowExecutionErrorWhenAlreadyStarted: true,
	}
	workflowInput := messages.UserAccountOrchestrationInput{
		Permissions:      make([]string, 0),
		AwaitingApproval: make([]string, 0),
	}

	run, err := h.c.ExecuteWorkflow(gc.Request.Context(), opts, "Orchestration", workflowInput)
	if err != nil {
		var ser *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &ser) {
			gc.Redirect(http.StatusSeeOther, "/users?flashUserAlreadyCreated="+workflowID)
			return
		}
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	if gc.Request.FormValue("make_user_approver") == "on" {
		updateOptions := client.UpdateWorkflowOptions{
			WorkflowID: run.GetID(),
			UpdateName: constants.CreateUserAccountUpdateHandlerName,
			Args: []interface{}{
				&messages.CreateUserAccountRequest{
					Permissions: []string{constants.PermissionTypeGrantPermissions},
				},
			},
			WaitForStage: client.WorkflowUpdateStageCompleted,
		}
		updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
		if err != nil {
			_ = gc.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		updateResponse := &messages.CreateUserAccountResponse{}
		err = updateHandle.Get(gc.Request.Context(), &updateResponse)
		if err != nil {
			_ = gc.AbortWithError(http.StatusInternalServerError, err)
			return
		}
	}
	gc.Redirect(http.StatusSeeOther, "/users?flashUserCreated="+workflowID)
}

func (h Handler) POSTDeleteUser(gc *gin.Context) {
	if gc.PostForm("username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "username required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("username"),
		UpdateName: constants.DeleteUserAccountUpdateHandlerName,
		Args: []interface{}{
			&messages.DeleteUserAccountRequest{},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := &messages.DeleteUserAccountResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}

func (h Handler) POSTRequestPermission(gc *gin.Context) {
	if gc.PostForm("username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "username required and missing")
		return
	}
	if gc.PostForm("permission_type") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "permission_type required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("username"),
		UpdateName: constants.AddUserPermissionUpdateHandlerName,
		Args: []interface{}{
			&messages.AddUserPermissionRequest{
				Permission: gc.PostForm("permission_type"),
			},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := &messages.AddUserPermissionResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := fmt.Sprintf("/users?permissionRequested=%s&username=%s", gc.PostForm("permission_type"),
		gc.PostForm("username"))
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}

func (h Handler) POSTUndoDeleteUser(gc *gin.Context) {
	if gc.PostForm("username") == "" {
		gc.AbortWithStatusJSON(http.StatusBadRequest, "username required and missing")
		return
	}
	updateOptions := client.UpdateWorkflowOptions{
		WorkflowID: gc.PostForm("username"),
		UpdateName: constants.UndoDeleteUserAccountUpdateHandlerName,
		Args: []interface{}{
			&messages.UndoDeleteUserAccountRequest{},
		},
		WaitForStage: client.WorkflowUpdateStageCompleted,
	}
	updateHandle, err := h.c.UpdateWorkflow(gc.Request.Context(), updateOptions)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	updateResponse := &messages.UndoDeleteUserAccountResponse{}
	err = updateHandle.Get(gc.Request.Context(), &updateResponse)
	if err != nil {
		_ = gc.AbortWithError(http.StatusInternalServerError, err)
		return
	}
	redirectRoute := "/user?id=" + gc.PostForm("username")
	gc.Redirect(http.StatusSeeOther, redirectRoute)
}
