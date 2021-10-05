package core

import (
	"groups/core/model"
)

//Services exposes APIs for the driver adapters
type Services interface {
	GetVersion() string

	GetGroupCategories() ([]string, error)
	GetUserGroupMemberships(externalID string) ([]*model.Group, *model.User, error)

	GetGroupEntity(clientID string, id string) (*model.Group, error)
	GetGroupEntityByMembership(clientID string, membershipID string) (*model.Group, error)

	CreateGroup(clientID string, current model.User, title string, description *string, category string, tags []string, privacy string,
		creatorName string, creatorEmail string, creatorPhotoURL string, imageURL *string, webURL *string) (*string, *GroupError)
	UpdateGroup(clientID string, current *model.User, id string, category string, title string, privacy string, description *string,
		imageURL *string, webURL *string, tags []string, membershipQuestions []string) *GroupError
	DeleteGroup(clientID string, current *model.User, id string) error
	GetGroups(clientID string, current *model.User, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]map[string]interface{}, error)
	GetUserGroups(clientID string, current *model.User) ([]map[string]interface{}, error)
	GetGroup(clientID string, current *model.User, id string) (map[string]interface{}, error)

	CreatePendingMember(clientID string, current model.User, groupID string, name string, email string, photoURL string, memberAnswers []model.MemberAnswer) error
	DeletePendingMember(clientID string, current model.User, groupID string) error
	DeleteMember(clientID string, current model.User, groupID string) error

	ApplyMembershipApproval(clientID string, current model.User, membershipID string, approve bool, rejectReason string) error
	DeleteMembership(clientID string, current model.User, membershipID string) error
	UpdateMembership(clientID string, current model.User, membershipID string, status string) error

	GetEvents(clientID string, groupID string) ([]model.Event, error)
	CreateEvent(clientID string, current model.User, eventID string, groupID string) error
	DeleteEvent(clientID string, current model.User, eventID string, groupID string) error

	GetPosts(clientID string, current *model.User, groupID string, offset *int64, limit *int64, order *string) ([]*model.Post, error)
	CreatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error)
	UpdatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error)
	DeletePost(clientID string, current *model.User, groupID string, postID string) error
}

type servicesImpl struct {
	app *Application
}

func (s *servicesImpl) GetVersion() string {
	return s.app.getVersion()
}

func (s *servicesImpl) GetGroupCategories() ([]string, error) {
	return s.app.getGroupCategories()
}
func (s *servicesImpl) GetUserGroupMemberships(externalID string) ([]*model.Group, *model.User, error) {
	return s.app.getUserGroupMemberships(externalID)
}

func (s *servicesImpl) GetGroupEntity(clientID string, id string) (*model.Group, error) {
	return s.app.getGroupEntity(clientID, id)
}

func (s *servicesImpl) GetGroupEntityByMembership(clientID string, membershipID string) (*model.Group, error) {
	return s.app.getGroupEntityByMembership(clientID, membershipID)
}

func (s *servicesImpl) CreateGroup(clientID string, current model.User, title string, description *string, category string, tags []string, privacy string,
	creatorName string, creatorEmail string, creatorPhotoURL string, imageURL *string, webURL *string) (*string, *GroupError) {
	return s.app.createGroup(clientID, current, title, description, category, tags, privacy, creatorName, creatorEmail, creatorPhotoURL,
		imageURL, webURL)
}

func (s *servicesImpl) UpdateGroup(clientID string, current *model.User, id string, category string, title string, privacy string, description *string,
	imageURL *string, webURL *string, tags []string, membershipQuestions []string) *GroupError {
	return s.app.updateGroup(clientID, current, id, category, title, privacy, description, imageURL, webURL, tags, membershipQuestions)
}

func (s *servicesImpl) DeleteGroup(clientID string, current *model.User, id string) error {
	return s.app.deleteGroup(clientID, current, id)
}

func (s *servicesImpl) GetGroups(clientID string, current *model.User, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]map[string]interface{}, error) {
	return s.app.getGroups(clientID, current, category, privacy, title, offset, limit, order)
}

func (s *servicesImpl) GetUserGroups(clientID string, current *model.User) ([]map[string]interface{}, error) {
	return s.app.getUserGroups(clientID, current)
}

func (s *servicesImpl) GetGroup(clientID string, current *model.User, id string) (map[string]interface{}, error) {
	return s.app.getGroup(clientID, current, id)
}

func (s *servicesImpl) CreatePendingMember(clientID string, current model.User, groupID string, name string, email string, photoURL string, memberAnswers []model.MemberAnswer) error {
	return s.app.createPendingMember(clientID, current, groupID, name, email, photoURL, memberAnswers)
}

func (s *servicesImpl) DeletePendingMember(clientID string, current model.User, groupID string) error {
	return s.app.deletePendingMember(clientID, current, groupID)
}

func (s *servicesImpl) DeleteMember(clientID string, current model.User, groupID string) error {
	return s.app.deleteMember(clientID, current, groupID)
}

func (s *servicesImpl) ApplyMembershipApproval(clientID string, current model.User, membershipID string, approve bool, rejectReason string) error {
	return s.app.applyMembershipApproval(clientID, current, membershipID, approve, rejectReason)
}

func (s *servicesImpl) DeleteMembership(clientID string, current model.User, membershipID string) error {
	return s.app.deleteMembership(clientID, current, membershipID)
}

func (s *servicesImpl) UpdateMembership(clientID string, current model.User, membershipID string, status string) error {
	return s.app.updateMembership(clientID, current, membershipID, status)
}

func (s *servicesImpl) GetEvents(clientID string, groupID string) ([]model.Event, error) {
	return s.app.getEvents(clientID, groupID)
}

func (s *servicesImpl) CreateEvent(clientID string, current model.User, eventID string, groupID string) error {
	return s.app.createEvent(clientID, current, eventID, groupID)
}

func (s *servicesImpl) DeleteEvent(clientID string, current model.User, eventID string, groupID string) error {
	return s.app.deleteEvent(clientID, current, eventID, groupID)
}

func (s *servicesImpl) GetPosts(clientID string, current *model.User, groupID string, offset *int64, limit *int64, order *string) ([]*model.Post, error) {
	return s.app.getPosts(clientID, current, groupID, offset, limit, order)
}

func (s *servicesImpl) CreatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error) {
	return s.app.createPost(clientID, current, post)
}

func (s *servicesImpl) UpdatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error) {
	return s.app.updatePost(clientID, current, post)
}

func (s *servicesImpl) DeletePost(clientID string, current *model.User, groupID string, postID string) error {
	return s.app.deletePost(clientID, current, groupID, postID)
}

//Administration exposes administration APIs for the driver adapters
type Administration interface {
	GetGroups(clientID string, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]model.Group, error)
}

type administrationImpl struct {
	app *Application
}

func (s *administrationImpl) GetGroups(clientID string, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]model.Group, error) {
	return s.app.getGroupsUnprotected(clientID, category, privacy, title, offset, limit, order)
}

//Storage is used by core to storage data - DB storage adapter, file storage adapter etc
type Storage interface {
	SetStorageListener(storageListener StorageListener)

	FindUser(clientID string, id string, external bool) (*model.User, error)
	CreateUser(clientID string, id string, externalID string, email string, isMemberOf *[]string) (*model.User, error)
	SaveUser(clientID string, user *model.User) error

	ReadAllGroupCategories() ([]string, error)
	FindUserGroupsMemberships(externalID string) ([]*model.Group, *model.User, error)

	CreateGroup(clientID string, title string, description *string, category string, tags []string, privacy string,
		creatorUserID string, creatorName string, creatorEmail string, creatorPhotoURL string, imageURL *string, webURL *string) (*string, *GroupError)
	UpdateGroup(clientID string, id string, category string, title string, privacy string, description *string,
		imageURL *string, webURL *string, tags []string, membershipQuestions []string) *GroupError
	DeleteGroup(clientID string, id string) error
	FindGroup(clientID string, id string) (*model.Group, error)
	FindGroupByMembership(clientID string, membershipID string) (*model.Group, error)
	FindGroups(clientID string, category *string, privacy *string, title *string, offset *int64, limit *int64, order *string) ([]model.Group, error)
	FindUserGroups(clientID string, userID string) ([]model.Group, error)

	CreatePendingMember(clientID string, groupID string, userID string, name string, email string, photoURL string, memberAnswers []model.MemberAnswer) error
	DeletePendingMember(clientID string, groupID string, userID string) error
	DeleteMember(clientID string, groupID string, userID string) error

	ApplyMembershipApproval(clientID string, membershipID string, approve bool, rejectReason string) error
	DeleteMembership(clientID string, currentUserID string, membershipID string) error
	UpdateMembership(clientID string, currentUserID string, membershipID string, status string) error

	FindEvents(clientID string, groupID string) ([]model.Event, error)
	CreateEvent(clientID string, eventID string, groupID string) error
	DeleteEvent(clientID string, eventID string, groupID string) error

	FindPosts(clientID string, current *model.User, groupID string, offset *int64, limit *int64, order *string) ([]*model.Post, error)
	FindPost(clientID string, current *model.User, groupID string, postID string, skipMembershipCheck bool) (*model.Post, error)
	FindPostsByParentID(clientID string, current *model.User, groupID string, parentID string, skipMembershipCheck bool, recursive bool, order *string) ([]*model.Post, error)
	CreatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error)
	UpdatePost(clientID string, current *model.User, post *model.Post) (*model.Post, error)
	DeletePost(clientID string, current *model.User, groupID string, postID string) error
}

//StorageListener listenes for change data storage events
type StorageListener interface {
	OnConfigsChanged()
}

type storageListenerImpl struct {
	app *Application
}

func (a *storageListenerImpl) OnConfigsChanged() {
	//do nothing for now
}
