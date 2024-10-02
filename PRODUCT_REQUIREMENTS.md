# Users & Permissions Product Requirements Doc

${BUSINESS} wants a user and permissions system. It is crucial that we support 
the minimum viable requirements outlined in this document to hit our quarterly
goals. 

## Open Questions 
* Given a dedicated full-time senior engineer how much effort is this to develop and test?
* What infrastructure components are required to build?
* Imagine there is an issue when the application is deployed to production
  * Given the infrastructure components: what steps are needed to troubleshoot?

## Requirements
1. Create a user
   1. Must be unique even if duplicate requests come in.
   2. Some users should be admins and be able to approve permissions for other users.
2. Delete a user
   1. Deletion should be a soft-delete for a given time window: once the time window passes the deletion is final.
3. Search for users with a given permission
   1. Should scale and be performant (<1s) even given a large (>1M) number of users.
   2. Eventual consistency on the search view is ok. 
4. Request a permission
5. Approve a permission
   1. Require approver to have authority to grant permissions.
6. Undo deletion
   1. A user should be able to undo the deletion of a user within the undo time window.
7. Reliability >99%
8. View all users
   1. Eventual consistency on the search view is ok.
9. View user details for a given user
   1. Remaining time to undo a deletion
   2. Permissions that have been approved
   3. Permissions requested
10. Extensible
    1. Support adding additional permission types within a three business day time frame
    2. Support the ability to add new UI clients within a single sprint 1
11. Auditable
    1. Every interaction with a user listed in this document and all future interactions need to be auditable
