package core

import "groups/core/model"

//Application represents the core application code based on hexagonal architecture
type Application struct {
	version string
	build   string

	Services       Services       //expose to the drivers adapters
	Administration Administration //expose to the drivrs adapters

	storage Storage
}

//Start starts the core part of the application
func (app *Application) Start() {
	//set storage listener
	storageListener := storageListenerImpl{app: app}
	app.storage.SetStorageListener(&storageListener)
}

//FindUser finds an user for the provided external id
func (app *Application) FindUser(externalID string) (*model.User, error) {
	user, err := app.storage.FindUser(externalID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

//CreateUser creates an user
func (app *Application) CreateUser(externalID string, email string, isMemberOf *[]string) (*model.User, error) {
	user, err := app.storage.CreateUser(externalID, email, isMemberOf)
	if err != nil {
		return nil, err
	}
	return user, nil
}

//UpdateUser updates the user
func (app *Application) UpdateUser(user *model.User) error {
	err := app.storage.SaveUser(user)
	if err != nil {
		return err
	}
	return nil
}

//NewApplication creates new Application
func NewApplication(version string, build string, storage Storage) *Application {

	application := Application{version: version, build: build, storage: storage}

	//add the drivers ports/interfaces
	application.Services = &servicesImpl{app: &application}
	application.Administration = &administrationImpl{app: &application}

	return &application
}
