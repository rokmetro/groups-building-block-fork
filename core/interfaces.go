package core

import (
	"groups/core/model"
)

//Services exposes APIs for the driver adapters
type Services interface {
	GetVersion() string

	GetGroupEntity(id string) (*model.Group, error)
	GetGroupEntityByMembership(membershipID string) (*model.Group, error)

	GetGroupCategories() ([]string, error)

	CreateGroup(current model.User, title string, description *string, category string, tags []string, privacy string,
		creatorName string, creatorEmail string, creatorPhotoURL string) (*string, error)
	UpdateGroup(current *model.User, id string, category string, title string, privacy string, description *string,
		imageURL *string, webURL *string, tags []string, membershipQuestions []string) error
	GetGroups(current *model.User, category *string) ([]map[string]interface{}, error)
	GetUserGroups(current *model.User) ([]map[string]interface{}, error)
	GetGroup(current *model.User, id string) (map[string]interface{}, error)

	CreatePendingMember(current model.User, groupID string, name string, email string, photoURL string, memberAnswers []model.MemberAnswer) error
	DeletePendingMember(current model.User, groupID string) error

	ApplyMembershipApproval(current model.User, membershipID string, approve bool, rejectReason string) error
}

type servicesImpl struct {
	app *Application
}

func (s *servicesImpl) GetVersion() string {
	return s.app.getVersion()
}

func (s *servicesImpl) GetGroupEntity(id string) (*model.Group, error) {
	return s.app.getGroupEntity(id)
}

func (s *servicesImpl) GetGroupEntityByMembership(membershipID string) (*model.Group, error) {
	return s.app.getGroupEntityByMembership(membershipID)
}

func (s *servicesImpl) GetGroupCategories() ([]string, error) {
	return s.app.getGroupCategories()
}

func (s *servicesImpl) CreateGroup(current model.User, title string, description *string, category string, tags []string, privacy string,
	creatorName string, creatorEmail string, creatorPhotoURL string) (*string, error) {
	return s.app.createGroup(current, title, description, category, tags, privacy, creatorName, creatorEmail, creatorPhotoURL)
}

func (s *servicesImpl) UpdateGroup(current *model.User, id string, category string, title string, privacy string, description *string,
	imageURL *string, webURL *string, tags []string, membershipQuestions []string) error {
	return s.app.updateGroup(current, id, category, title, privacy, description, imageURL, webURL, tags, membershipQuestions)
}

func (s *servicesImpl) GetGroups(current *model.User, category *string) ([]map[string]interface{}, error) {
	return s.app.getGroups(current, category)
}

func (s *servicesImpl) GetUserGroups(current *model.User) ([]map[string]interface{}, error) {
	return s.app.getUserGroups(current)
}

func (s *servicesImpl) GetGroup(current *model.User, id string) (map[string]interface{}, error) {
	return s.app.getGroup(current, id)
}

func (s *servicesImpl) CreatePendingMember(current model.User, groupID string, name string, email string, photoURL string, memberAnswers []model.MemberAnswer) error {
	return s.app.createPendingMember(current, groupID, name, email, photoURL, memberAnswers)
}

func (s *servicesImpl) DeletePendingMember(current model.User, groupID string) error {
	return s.app.deletePendingMember(current, groupID)
}

func (s *servicesImpl) ApplyMembershipApproval(current model.User, membershipID string, approve bool, rejectReason string) error {
	return s.app.applyMembershipApproval(current, membershipID, approve, rejectReason)
}

//Administration exposes administration APIs for the driver adapters
type Administration interface {
	GetTODO() error
}

type administrationImpl struct {
	app *Application
}

func (s *administrationImpl) GetTODO() error {
	return s.app.getTODO()
}

//Storage is used by core to storage data - DB storage adapter, file storage adapter etc
type Storage interface {
	SetStorageListener(storageListener StorageListener)

	FindUser(externalID string) (*model.User, error)
	CreateUser(externalID string, email string, isMemberOf *[]string) (*model.User, error)
	SaveUser(user *model.User) error

	ReadAllGroupCategories() ([]string, error)

	CreateGroup(title string, description *string, category string, tags []string, privacy string,
		creatorUserID string, creatorName string, creatorEmail string, creatorPhotoURL string) (*string, error)
	UpdateGroup(id string, category string, title string, privacy string, description *string,
		imageURL *string, webURL *string, tags []string, membershipQuestions []string) error
	FindGroup(id string) (*model.Group, error)
	FindGroupByMembership(membershipID string) (*model.Group, error)
	FindGroups(category *string) ([]model.Group, error)
	FindUserGroups(userID string) ([]model.Group, error)

	CreatePendingMember(groupID string, userID string, name string, email string, photoURL string, memberAnswers []model.MemberAnswer) error
	DeletePendingMember(groupID string, userID string) error

	ApplyMembershipApproval(membershipID string, approve bool, rejectReason string) error
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
