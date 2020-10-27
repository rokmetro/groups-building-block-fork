package web

import (
	"context"
	"fmt"
	"groups/core"
	"groups/core/model"
	"groups/utils"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/syncmap"
	"gopkg.in/ericchiang/go-oidc.v2"
)

//Auth handler
type Auth struct {
	apiKeysAuth *APIKeysAuth
	idTokenAuth *IDTokenAuth
}

//Start starts the auth module
func (auth *Auth) Start() error {
	auth.idTokenAuth.start()

	return nil
}

func (auth *Auth) apiKeyCheck(w http.ResponseWriter, r *http.Request) bool {
	apiKey := auth.getAPIKey(r)
	return auth.apiKeysAuth.check(apiKey, w)
}

func (auth *Auth) idTokenCheck(w http.ResponseWriter, r *http.Request) *model.User {
	idToken := auth.getIDToken(r)
	return auth.idTokenAuth.check(idToken, w)
}

func (auth *Auth) mixedCheck(w http.ResponseWriter, r *http.Request) (bool, *model.User) {
	//first check for id token
	idToken := auth.getIDToken(r)
	if idToken != nil && len(*idToken) > 0 {
		authenticated := false
		user := auth.idTokenAuth.check(idToken, w)
		if user != nil {
			authenticated = true
		}
		return authenticated, user
	}

	//check api key
	apiKey := auth.getAPIKey(r)
	if apiKey != nil && len(*apiKey) > 0 {
		return auth.apiKeysAuth.check(apiKey, w), nil
	}

	//neither id token nor api key - so bad request
	log.Println("400 - Bad Request")
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad Request"))
	return false, nil
}

func (auth *Auth) getAPIKey(r *http.Request) *string {
	apiKey := r.Header.Get("ROKWIRE-API-KEY")
	if len(apiKey) == 0 {
		return nil
	}
	return &apiKey
}

func (auth *Auth) getIDToken(r *http.Request) *string {
	// get the token from the request
	authorizationHeader := r.Header.Get("Authorization")
	if len(authorizationHeader) <= 0 {
		return nil
	}
	splitAuthorization := strings.Fields(authorizationHeader)
	if len(splitAuthorization) != 2 {
		return nil
	}
	// expected - Bearer 1234
	if splitAuthorization[0] != "Bearer" {
		return nil
	}
	idToken := splitAuthorization[1]
	return &idToken
}

//NewAuth creates new auth handler
func NewAuth(app *core.Application, appKeys []string, oidcProvider string, oidcClientID string) *Auth {
	apiKeysAuth := newAPIKeysAuth(appKeys)
	idTokenAuth := newIDTokenAuth(app, oidcProvider, oidcClientID)

	auth := Auth{apiKeysAuth: apiKeysAuth, idTokenAuth: idTokenAuth}
	return &auth
}

/////////////////////////////////////

//APIKeysAuth entity
type APIKeysAuth struct {
	appKeys []string
}

func (auth *APIKeysAuth) check(apiKey *string, w http.ResponseWriter) bool {
	//check if there is api key in the header
	if apiKey == nil || len(*apiKey) == 0 {
		//no key, so return 400
		log.Println(fmt.Sprintf("400 - Bad Request"))

		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
		return false
	}

	//check if the api key is one of the listed
	appKeys := auth.appKeys
	exist := false
	for _, element := range appKeys {
		if element == *apiKey {
			exist = true
			break
		}
	}
	if !exist {
		//not exist, so return 401
		log.Println(fmt.Sprintf("401 - Unauthorized for key %s", *apiKey))

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return false
	}
	return true
}

//NewAPIKeysAuth creates new api keys auth
func newAPIKeysAuth(appKeys []string) *APIKeysAuth {
	auth := APIKeysAuth{appKeys}
	return &auth
}

////////////////////////////////////

type userData struct {
	UIuceduUIN        *string   `json:"uiucedu_uin"`
	Sub               *string   `json:"sub"`
	Email             *string   `json:"email"`
	UIuceduIsMemberOf *[]string `json:"uiucedu_is_member_of"`
}

type cacheUser struct {
	user      *model.User
	lastUsage time.Time
}

//IDTokenAuth entity
type IDTokenAuth struct {
	app *core.Application

	idTokenVerifier *oidc.IDTokenVerifier

	cachedUsers     *syncmap.Map //cache users while active - 5 minutes timeout
	cachedUsersLock *sync.RWMutex
}

func (auth *IDTokenAuth) start() {
	go auth.cleanCacheUser()
}

//cleanChacheUser cleans all users from the cache with no activity > 5 minutes
func (auth *IDTokenAuth) cleanCacheUser() {
	log.Println("IDTokenAuth -> cleanCacheUser -> start")

	toRemove := []string{}

	//find all users to remove - more than 5 minutes period from their last usage
	now := time.Now().Unix()
	auth.cachedUsers.Range(func(key, value interface{}) bool {
		cacheUser, ok := value.(*cacheUser)
		if !ok {
			return false //break the iteration
		}
		externalID, ok := key.(string)
		if !ok {
			return false //break the iteration
		}

		difference := now - cacheUser.lastUsage.Unix()
		//5 minutes
		if difference > 300 {
			toRemove = append(toRemove, externalID)
		}

		// this will continue iterating
		return true
	})

	//remove the selected ones
	count := len(toRemove)
	if count > 0 {
		log.Printf("IDTokenAuth -> cleanCacheUser -> %d items to remove\n", count)

		for _, key := range toRemove {
			auth.deleteCacheUser(key)
		}
	} else {
		log.Println("IDTokenAuth -> cleanCacheUser -> nothing to remove")
	}

	nextLoad := time.Minute * 5
	log.Printf("IDTokenAuth -> cleanCacheUser() -> next exec after %s\n", nextLoad)
	timer := time.NewTimer(nextLoad)
	<-timer.C
	log.Println("IDTokenAuth -> cleanCacheUser() -> timer expired")

	auth.cleanCacheUser()
}

func (auth *IDTokenAuth) check(token *string, w http.ResponseWriter) *model.User {
	//1. Check if there is a token
	if token == nil || len(*token) == 0 {
		auth.responseBadRequest(w)
		return nil
	}
	rawIDToken := *token

	//2. Validate the token
	idToken, err := auth.idTokenVerifier.Verify(context.Background(), rawIDToken)
	if err != nil {
		log.Printf("error validating token - %s\n", err)

		auth.responseUnauthorized(rawIDToken, w)
		return nil
	}

	//3. Get the user data from the token
	var userData userData
	if err := idToken.Claims(&userData); err != nil {
		log.Printf("error getting user data from token - %s\n", err)

		auth.responseUnauthorized(rawIDToken, w)
		return nil
	}
	//we must have UIuceduUIN
	if userData.UIuceduUIN == nil {
		log.Printf("missing uiuceuin data in the token - %s\n", err)

		auth.responseUnauthorized(rawIDToken, w)
		return nil
	}

	//4. Get the user for the provided external id.
	user, err := auth.getUser(userData)
	if err != nil {
		log.Printf("error getting an user for external id - %s\n", err)

		auth.responseInternalServerError(w)
		return nil
	}
	if user == nil {
		log.Printf("for some reasons the user for external id - %s is nil\n", err)

		auth.responseInternalServerError(w)
		return nil
	}

	//5. Update the user if needed
	user, err = auth.updateUserIfNeeded(*user, userData)
	if err != nil {
		log.Printf("error updating an user for external id - %s\n", err)

		auth.responseInternalServerError(w)
		return nil
	}

	//6. Return the user
	return user
}

func (auth *IDTokenAuth) updateUserIfNeeded(current model.User, userData userData) (*model.User, error) {
	currentList := current.IsMemberOf
	newList := userData.UIuceduIsMemberOf

	isEqual := utils.EqualPointers(currentList, newList)
	if !isEqual {
		log.Println("updateUserIfNeeded -> need to update user")

		//1. remove it from the cache
		auth.deleteCacheUser(current.ExternalID)

		//2. update it
		current.IsMemberOf = userData.UIuceduIsMemberOf
		err := auth.app.UpdateUser(&current)
		if err != nil {
			return nil, err
		}
	}

	return &current, nil
}

func (auth *IDTokenAuth) getCachedUser(externalID string) *cacheUser {
	auth.cachedUsersLock.RLock()
	defer auth.cachedUsersLock.RUnlock()

	var cachedUser *cacheUser //to return

	item, _ := auth.cachedUsers.Load(externalID)
	if item != nil {
		cachedUser = item.(*cacheUser)
	}

	//keep the last get time
	if cachedUser != nil {
		cachedUser.lastUsage = time.Now()
		auth.cachedUsers.Store(externalID, cachedUser)
	}

	return cachedUser
}

func (auth *IDTokenAuth) cacheUser(externalID string, user *model.User) {
	auth.cachedUsersLock.RLock()

	cacheUser := &cacheUser{user: user, lastUsage: time.Now()}
	auth.cachedUsers.Store(externalID, cacheUser)

	auth.cachedUsersLock.RUnlock()
}

func (auth *IDTokenAuth) deleteCacheUser(externalID string) {
	auth.cachedUsersLock.RLock()

	auth.cachedUsers.Delete(externalID)

	auth.cachedUsersLock.RUnlock()
}

func (auth *IDTokenAuth) getUser(userData userData) (*model.User, error) {
	var err error

	//1. First check if cached
	cachedUser := auth.getCachedUser(*userData.UIuceduUIN)
	if cachedUser != nil {
		return cachedUser.user, nil
	}

	//2. Check if we have a such user in the application
	user, err := auth.app.FindUser(*userData.UIuceduUIN)
	if err != nil {
		log.Printf("error finding an for external id - %s\n", err)
		return nil, err
	}
	if user != nil {
		//cache it
		auth.cacheUser(*userData.UIuceduUIN, user)
		return user, nil
	}

	//3. This is the first call for the user, so we need to create it
	user, err = auth.app.CreateUser(*userData.UIuceduUIN, *userData.Email, userData.UIuceduIsMemberOf)
	if err != nil {
		log.Printf("error creating an user - %s\n", err)
		return nil, err
	}
	//cache it
	auth.cacheUser(*userData.UIuceduUIN, user)
	return user, nil
}

func (auth *IDTokenAuth) responseBadRequest(w http.ResponseWriter) {
	log.Println(fmt.Sprintf("400 - Bad Request"))

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad Request"))
}

func (auth *IDTokenAuth) responseUnauthorized(token string, w http.ResponseWriter) {
	log.Println(fmt.Sprintf("401 - Unauthorized for token %s", token))

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

func (auth *IDTokenAuth) responseInternalServerError(w http.ResponseWriter) {
	log.Println(fmt.Sprintf("500 - Internal Server Error"))

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
}

//newIDTokenAuth creates new id token auth
func newIDTokenAuth(app *core.Application, oidcProvider string, oidcClientID string) *IDTokenAuth {
	provider, err := oidc.NewProvider(context.Background(), oidcProvider)
	if err != nil {
		log.Fatalln(err)
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{ClientID: oidcClientID})

	cacheUsers := &syncmap.Map{}
	lock := &sync.RWMutex{}

	auth := IDTokenAuth{app: app, idTokenVerifier: idTokenVerifier,
		cachedUsers: cacheUsers, cachedUsersLock: lock}
	return &auth
}
