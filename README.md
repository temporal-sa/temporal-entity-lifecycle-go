# Entity Workflow Demo

## What is an Entity?
> In domain-driven design, an entity is a representation of an object 
in the domain. It is defined by its identity, rather than its attributes. 
It encapsulates the state of that object through its attributes, including 
the aggregation of other entities, and it defines any operations that might be 
performed on the entity.

*Source*: https://www.cockroachlabs.com/blog/relational-database-entities/

## Purpose of this demo
The purpose of this demo is to illustrate some of the interesting properties
of the Entity Lifecycle Pattern (aka Entity Workflows) using the Temporal Go SDK.
This is meant to be a mostly thorough pairing with the ["Long-running workflow" slide deck](
https://docs.google.com/presentation/d/1wuEwx0U2AyKpHPRtg_e7iDXyOuZ5mvAK0y0DNsLy-C0/edit?usp=sharing)


Although this demo is proposing the use of a `UserAccount` this entity was chosen 
as it is a familiar concept for many persons involved in software. The concepts
used here can be applied to any use case where one wants to model an entity as 
code, for example:

* Provisioning and managing compute infrastructure
* Marketing campaigns (e.g. drip emails/sms/user journeys)
* Human-in-the-loop processes (e.g. approving an expense report, order fulfillment)
* Loyalty programs
* Customer onboarding
* Loan applications
* A virtual shopping cart
* Chat sessions with an LLM (e.g. Gemini, ChatGPT)


## Running the demo
Environment variables needed for `go run` commands:
```
TEMPORAL_CLIENT_KEY_PATH 
TEMPORAL_CLIENT_CERT_PATH 
TEMPORAL_CLIENT_HOSTPORT
TEMPORAL_CLIENT_NAMESPACE
```
1. Review `PRODUCT_REQUIREMENTS.md` with the audience
2. Open a terminal window and run: `go run entity-demo/cmd/worker/worker.go`
3. Open a terminal window and run: `go run entity-demo/cmd/web/web.go`
4. In browser visit `localhost:8081/create_user`
5. Create a user, make that user an approver
6. Create user with the same name as the previous step: only the first user has been created!
7. Create a new user with a new name
8. View each user profile and review their permissions: the first user should have `grant_permisssions` and the second should not have any
9. Request the `read_files` permission for the second user
10. Approve the `read_files` permission for the second user with the first user being the approver
11. Delete the second user via the button in their profile view
12. Refresh the page to see the time window to undo deletion is ticking down
13. Undo the delete
14. Search for users w/ each permission type (Users view)
15. View Temporal UI and review the event history for each workflow, discuss which Temporal primitives are used
16. Code walkthrough: 
    1. `orchestrations/user_account_handler.go`
    2. `user_account_state/user_account_state.go` 
    3. `orchestrations/activity_handler/activity_handler.go`
17. Review tests:
    1. `orchestrations/user_account_handler_test.go`
18. Show in the Temporal UI where to download replay history, mention that you can also use the SDK or CLI to download histories
19. Perform a replay test (`orchestrations/user_account_handler_test.go:Test_Orchestration_ReplayHistory`)
20. Uncomment `user_account_state/user_account_state.go:126-130` & rerun replay test: fails because of nondeterminism 
21. Uncomment `user_account_state/user_account_state.go:124-131` & rerun replay test: passes because of versioning

## Miscellaneous Notes
### Custom Search Attributes

To search in the Cloud UI: 
```
`permissions`="{my_permission_name}"
```