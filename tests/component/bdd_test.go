//go:build component
// +build component

package component

func (s *ComponentTestSuite) TestCreateUser() {
	_, when, then := s.gherkin()

	when().
		aCreateUserRequestIsIssued()

	then().
		theCreateUserResponseContainsAValidUser().
		listUsersContainsTheCreatedUser().
		anEventForTheUserCreationWillEventuallyBeProduced()
}


func (s *ComponentTestSuite) TestUpdateUser() {
	given, when, then := s.gherkin()

	given().
		anExistingUser()

	when().
		theUserGetsUpdated()

	then().
		theUpdateResponseReflectsTheUpdateOperation().
		listUsersContainsTheUpdatedUser().
		anEventForTheUserUpdateWillEventuallyBeProduced()
}

func (s *ComponentTestSuite) TestDeleteUser() {
	given, when, then := s.gherkin()

	given().
		anExistingUser()

	when().
		aUserDeletionRequestIsIssued()

	then().
		listUsersDoesNotContainTheUser().
		anEventForTheUserDeletionWillEventuallyBeProduced()
}