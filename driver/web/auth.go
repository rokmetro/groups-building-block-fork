package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/rokwire/logging-library-go/logs"
	"golang.org/x/sync/syncmap"
	"gopkg.in/ericchiang/go-oidc.v2"
	"groups/core"
	"groups/core/model"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/casbin/casbin"
	"github.com/rokwire/core-auth-library-go/authorization"
	"github.com/rokwire/core-auth-library-go/authservice"
	"github.com/rokwire/core-auth-library-go/authutils"
	"github.com/rokwire/core-auth-library-go/tokenauth"
)

//Auth handler
type Auth struct {
	apiKeysAuth  *APIKeysAuth
	idTokenAuth  *IDTokenAuth
	internalAuth *InternalAuth
	adminAuth    *AdminAuth

	supportedClients []string
}

func (auth *Auth) clientIDCheck(r *http.Request) (bool, string) {
	clientID := r.Header.Get("APP")
	if len(clientID) == 0 {
		clientID = "edu.illinois.rokwire"
	}

	//check if supported
	for _, s := range auth.supportedClients {
		if s == clientID {
			return true, clientID
		}
	}
	return false, ""
}

func (auth *Auth) apiKeyCheck(r *http.Request) (string, bool) {
	clientIDOK, clientID := auth.clientIDCheck(r)
	if !clientIDOK {
		return "", false
	}

	apiKey := auth.getAPIKey(r)
	authenticated := auth.apiKeysAuth.check(apiKey, r)

	return clientID, authenticated
}

func (auth *Auth) idTokenCheck(w http.ResponseWriter, r *http.Request) (string, *model.User) {
	clientIDOK, clientID := auth.clientIDCheck(r)
	if !clientIDOK {
		return "", nil
	}

	idToken := auth.getIDToken(r)
	user := auth.idTokenAuth.check(clientID, idToken, nil, r)
	return clientID, user
}

func (auth *Auth) customClientTokenCheck(w http.ResponseWriter, r *http.Request, allowedOIDCClientIDs []string) (string, *model.User) {
	clientIDOK, clientID := auth.clientIDCheck(r)
	if !clientIDOK {
		return "", nil
	}

	idToken := auth.getIDToken(r)
	user := auth.idTokenAuth.check(clientID, idToken, allowedOIDCClientIDs, r)
	return clientID, user
}

func (auth *Auth) internalAuthCheck(w http.ResponseWriter, r *http.Request) (string, bool) {
	clientIDOK, clientID := auth.clientIDCheck(r)
	if !clientIDOK {
		return "", false
	}

	internalAuthKey := auth.getInternalAPIKey(r)
	authenticated := auth.internalAuth.check(internalAuthKey, w)

	return clientID, authenticated
}

func (auth *Auth) mixedCheck(r *http.Request) (string, bool, *model.User) {
	//get client ID
	clientIDOK, clientID := auth.clientIDCheck(r)
	if !clientIDOK {
		return "", false, nil
	}

	//first check for id token
	idToken := auth.getIDToken(r)
	if idToken != nil && len(*idToken) > 0 {
		authenticated := false
		user := auth.idTokenAuth.check(clientID, idToken, nil, r)
		if user != nil {
			authenticated = true
		}
		return clientID, authenticated, user
	}

	//check api key
	apiKey := auth.getAPIKey(r)
	if apiKey != nil && len(*apiKey) > 0 {
		authenticated := auth.apiKeysAuth.check(apiKey, r)
		return clientID, authenticated, nil
	}
	return clientID, false, nil
}

func (auth *Auth) adminCheck(r *http.Request) (string, *model.User, bool) {
	clientIDOK, clientID := auth.clientIDCheck(r)
	if !clientIDOK {
		return "", nil, false
	}

	user, forbidden := auth.adminAuth.check(clientID, r)
	return clientID, user, forbidden
}

func (auth *Auth) getAPIKey(r *http.Request) *string {
	apiKey := r.Header.Get("ROKWIRE-API-KEY")
	if len(apiKey) == 0 {
		return nil
	}
	return &apiKey
}

func (auth *Auth) getInternalAPIKey(r *http.Request) *string {
	apiKey := r.Header.Get("ROKWIRE_GS_API_KEY")
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
func NewAuth(app *core.Application, host string, appKeys []string, internalAPIKeys []string, oidcProvider string, oidcClientID string, oidcExtendedClientIDs string,
	oidcAdminClientID string, oidcAdminWebClientID string, coreBBHost string, groupServiceURL string, adminAuthorization *casbin.Enforcer) *Auth {
	var tokenAuth *tokenauth.TokenAuth
	if coreBBHost != "" {
		serviceID := "groups"

		remoteConfig := authservice.RemoteAuthDataLoaderConfig{
			AuthServicesHost: coreBBHost,
		}

		// Instantiate a remote ServiceRegLoader to load auth service registration record from auth service
		serviceLoader, err := authservice.NewRemoteAuthDataLoader(remoteConfig, []string{coreBBHost + "/bbs/service-regs"}, logs.NewLogger("groupsbb", &logs.LoggerOpts{}))

		// Instantiate AuthService instance
		authService, err := authservice.NewAuthService(serviceID, groupServiceURL, serviceLoader)
		if err != nil {
			log.Fatalf("error instancing auth service: %s", err)
		}

		permissionAuth := authorization.NewCasbinStringAuthorization("driver/web/permissions_authorization_policy.csv")
		scopeAuth := authorization.NewCasbinScopeAuthorization("driver/web/scope_authorization_policy.csv", serviceID)

		// Instantiate TokenAuth instance to perform token validation
		tokenAuth, err = tokenauth.NewTokenAuth(true, authService, permissionAuth, scopeAuth)
		if err != nil {
			log.Fatalf("error instancing token auth: %s", err)
		}
	}

	apiKeysAuth := newAPIKeysAuth(appKeys, tokenAuth)
	idTokenAuth := newIDTokenAuth(app, oidcProvider, oidcClientID, oidcExtendedClientIDs, tokenAuth)
	internalAuth := newInternalAuth(internalAPIKeys)
	adminAuth := newAdminAuth(app, oidcProvider, oidcAdminClientID, oidcAdminWebClientID, tokenAuth, adminAuthorization)

	supportedClients := []string{"edu.illinois.rokwire", "edu.illinois.covid"}

	auth := Auth{apiKeysAuth: apiKeysAuth, idTokenAuth: idTokenAuth, internalAuth: internalAuth, adminAuth: adminAuth, supportedClients: supportedClients}
	return &auth
}

/////////////////////////////////////

//APIKeysAuth entity
type APIKeysAuth struct {
	appKeys []string

	coreTokenAuth *tokenauth.TokenAuth
}

func (auth *APIKeysAuth) check(apiKey *string, r *http.Request) bool {
	//check if there is api key in the header
	if apiKey == nil || len(*apiKey) == 0 {
		if auth.coreTokenAuth != nil {
			_, err := auth.coreTokenAuth.CheckRequestTokens(r)
			if err == nil {
				return true
			}
			return false
		}
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
		return false
	}
	return true
}

//NewAPIKeysAuth creates new api keys auth
func newAPIKeysAuth(appKeys []string, coreTokenAuth *tokenauth.TokenAuth) *APIKeysAuth {
	auth := APIKeysAuth{appKeys, coreTokenAuth}
	return &auth
}

////////////////////////////////////

type userData struct {
	UIuceduUIN        *string   `json:"uiucedu_uin"`
	Sub               *string   `json:"sub"`
	Aud               *string   `json:"aud"`
	Email             *string   `json:"email"`
	Name              *string   `json:"name"`
	UIuceduIsMemberOf *[]string `json:"uiucedu_is_member_of"`
}

////////////////////////////////////

//InternalAuth entity
type InternalAuth struct {
	appKeys []string
}

func (auth *InternalAuth) check(internalKey *string, w http.ResponseWriter) bool {
	//check if there is internal key in the header
	if internalKey == nil || len(*internalKey) == 0 {
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
		if element == *internalKey {
			exist = true
			break
		}
	}
	if !exist {
		//not exist, so return 401
		log.Println(fmt.Sprintf("401 - Unauthorized for key %s", *internalKey))

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
		return false
	}
	return true
}

//newInternalAuth creates new internal auth
func newInternalAuth(internalAPIKeys []string) *InternalAuth {
	auth := InternalAuth{appKeys: internalAPIKeys}
	return &auth
}

///////////////////////////////////

//IDTokenAuth entity
type IDTokenAuth struct {
	app *core.Application

	idTokenVerifier   *oidc.IDTokenVerifier
	appClientIDs      []string
	extendedClientIDs []string

	coreTokenAuth *tokenauth.TokenAuth

	cachedUsers            *syncmap.Map //cache users while active - 5 minutes timeout
	cachedUsersLock        *sync.RWMutex
	cachedUsersLockMapping map[string]*sync.Mutex
}

func (auth *IDTokenAuth) check(clientID string, token *string, allowedClientIDs []string, r *http.Request) *model.User {
	var data *userData
	var isCoreUser = false
	if auth.coreTokenAuth != nil {
		claims, err := auth.coreTokenAuth.CheckRequestTokens(r)
		if err == nil && claims != nil && !claims.Anonymous {
			err = auth.coreTokenAuth.AuthorizeRequestScope(claims, r)
			if err != nil {
				return nil
			}

			log.Printf("Authentication successful for user: %v", claims)
			permissions := strings.Split(claims.Permissions, ",")
			data = &userData{Sub: &claims.Subject, Email: &claims.Email, Name: &claims.Name,
				UIuceduIsMemberOf: &permissions, UIuceduUIN: &claims.UID}
			isCoreUser = true
		}
	}

	if data == nil {
		//1. Check if there is a token
		if token == nil || len(*token) == 0 {
			return nil
		}
		rawIDToken := *token

		//2. Validate the token
		idToken, err := auth.idTokenVerifier.Verify(context.Background(), rawIDToken)
		if err != nil {
			log.Printf("error validating token - %s\n", err)
			return nil
		}

		//3. Get the user data from the token
		if err := idToken.Claims(&data); err != nil {
			log.Printf("error getting user data from token - %s\n", err)
			return nil
		}

		if allowedClientIDs == nil {
			allowedClientIDs = auth.appClientIDs
		}

		validAud := false
		if data.Aud != nil {
			validAud = authutils.ContainsString(allowedClientIDs, *data.Aud)
		}
		if !validAud {
			log.Printf("invalid audience in token: expected %v, got %s\n", allowedClientIDs, *data.Aud)
			return nil
		}

		//we must have UIuceduUIN
		if data.UIuceduUIN == nil {
			log.Printf("missing uiuceuin data in the token - %s\n", err)
			return nil
		}
	}

	if data == nil {
		log.Println("nil user data")
		return nil
	}

	// 4. Use corebb user id or legacy user id
	// NOTE: In general we assume the corebb user is already refactored e.g the login API has been invoked at least once.
	// The difference would be only the user ID.
	var userID string
	if isCoreUser {
		userID = *data.Sub
	} else {
		persistedUser, err := auth.app.FindUser(clientID, data.UIuceduUIN, true)
		if err != nil {
			log.Printf("error retriving user (UIuceduUIN: %s): %s", *data.UIuceduUIN, err)
		}
		if persistedUser != nil {
			isCoreUser = persistedUser.IsCoreUser
			userID = persistedUser.ID
		} else {
			legacyUser, err := auth.app.CreateUser(clientID, uuid.NewString(), data.UIuceduUIN, data.Email)
			if err != nil {
				log.Printf("error creating legacy user (UIuceduUIN: %s): %s", *data.UIuceduUIN, err)
			}
			if legacyUser != nil {
				userID = legacyUser.ID
			}
		}
	}

	//5. Get the user for the provided external id.
	var name = ""
	var externalID = ""
	var email = ""
	if data.Name != nil {
		name = *data.Name
	}
	if data.UIuceduUIN != nil {
		externalID = *data.UIuceduUIN
	}
	if data.Email != nil {
		email = *data.Email
	}
	return &model.User{ID: userID, ClientID: clientID, ExternalID: externalID, Email: email, Name: name, IsCoreUser: isCoreUser}
}

func (auth *IDTokenAuth) responseBadRequest(w http.ResponseWriter) {
	log.Println("400 - Bad Request")

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad Request"))
}

func (auth *IDTokenAuth) responseUnauthorized(token string, w http.ResponseWriter) {
	log.Printf("401 - Unauthorized for token %s", token)

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

func (auth *IDTokenAuth) responseInternalServerError(w http.ResponseWriter) {
	log.Printf("500 - Internal Server Error")

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
}

//newIDTokenAuth creates new id token auth
func newIDTokenAuth(app *core.Application, oidcProvider string, appClientIDs string, extendedClientIDs string, coreTokenAuth *tokenauth.TokenAuth) *IDTokenAuth {
	provider, err := oidc.NewProvider(context.Background(), oidcProvider)
	if err != nil {
		log.Fatalln(err)
	}
	idTokenVerifier := provider.Verifier(&oidc.Config{SkipClientIDCheck: true})

	cacheUsers := &syncmap.Map{}
	lock := &sync.RWMutex{}

	appClientIDList := strings.Split(appClientIDs, ",")
	extendedClientIDList := strings.Split(extendedClientIDs, ",")

	auth := IDTokenAuth{app: app, idTokenVerifier: idTokenVerifier, coreTokenAuth: coreTokenAuth,
		appClientIDs: appClientIDList, extendedClientIDs: extendedClientIDList,
		cachedUsers: cacheUsers, cachedUsersLock: lock}
	return &auth
}

////////////////////////////////////

//AdminAuth entity
type AdminAuth struct {
	app *core.Application

	appVerifier    *oidc.IDTokenVerifier
	appClientID    string
	webAppVerifier *oidc.IDTokenVerifier
	webAppClientID string

	authorization *casbin.Enforcer

	coreTokenAuth *tokenauth.TokenAuth

	cachedUsers     *syncmap.Map //cache users while active - 5 minutes timeout
	cachedUsersLock *sync.RWMutex
}

func (auth *AdminAuth) start() {

}

func (auth *AdminAuth) check(clientID string, r *http.Request) (*model.User, bool) {
	var data *userData
	var isCoreUser = false

	if auth.coreTokenAuth != nil {
		claims, err := auth.coreTokenAuth.CheckRequestTokens(r)
		if err == nil && claims != nil && !claims.Anonymous {
			err = auth.coreTokenAuth.AuthorizeRequestPermissions(claims, r)
			if err != nil {
				return nil, true
			}

			permissions := strings.Split(claims.Permissions, ",")
			data = &userData{Sub: &claims.Subject, Email: &claims.Email, Name: &claims.Name,
				UIuceduIsMemberOf: &permissions, UIuceduUIN: &claims.UID}
			isCoreUser = true
		}
	}

	if data == nil {
		//1. Get the token from the request
		rawIDToken, tokenType, err := auth.getIDToken(r)
		if err != nil {
			return nil, false
		}

		//3. Validate the token
		idToken, err := auth.verify(*rawIDToken, *tokenType)
		if err != nil {
			log.Printf("error validating token - %s\n", err)
			return nil, false
		}

		//4. Get the user data from the token
		if err := idToken.Claims(&data); err != nil {
			log.Printf("error getting user data from token - %s\n", err)
			return nil, false
		}

		//we must have UIuceduUIN
		if data.UIuceduUIN == nil {
			log.Printf("error - missing uiuceuin data in the token - %s\n", err)
			return nil, false
		}

		obj := r.URL.Path // the resource that is going to be accessed.
		act := r.Method   // the operation that the user performs on the resource.

		hasAccess := false
		for _, s := range *data.UIuceduIsMemberOf {
			hasAccess = auth.authorization.Enforce(s, obj, act)
			if hasAccess {
				break
			}
		}

		if !hasAccess {
			log.Printf("Access control error - UIN: %s is trying to apply %s operation for %s\n", *data.UIuceduUIN, act, obj)
			return nil, true
		}
	}

	if data == nil {
		log.Println("nil user data")
		return nil, false
	}

	var name = ""
	var externalID = ""
	var email = ""
	var userID = ""
	if data.Sub != nil {
		userID = *data.Sub
	}
	if data.Name != nil {
		name = *data.Name
	}
	if data.UIuceduUIN != nil {
		externalID = *data.UIuceduUIN
	}
	if data.Email != nil {
		email = *data.Email
	}
	return &model.User{ID: userID, ClientID: clientID, ExternalID: externalID, Email:email, Name: name, IsCoreUser: isCoreUser}, false
}

//gets the token from the request - as cookie or as Authorization header.
//returns the id token and its type - mobile or web. If the token is taken by the cookie it is web otherwise it is mobile
func (auth *AdminAuth) getIDToken(r *http.Request) (*string, *string, error) {
	var tokenType string

	//1. Check if there is a cookie
	cookie, err := r.Cookie("rwa-at-data")
	if err == nil && cookie != nil && len(cookie.Value) > 0 {
		//there is a cookie
		tokenType = "web"
		return &cookie.Value, &tokenType, nil
	}

	//2. Check if there is a token in the Authorization header
	authorizationHeader := r.Header.Get("Authorization")
	if len(authorizationHeader) <= 0 {
		return nil, nil, errors.New("error getting Authorization header")
	}
	splitAuthorization := strings.Fields(authorizationHeader)
	if len(splitAuthorization) != 2 {
		return nil, nil, errors.New("error processing the Authorization header")
	}
	// expected - Bearer 1234
	if splitAuthorization[0] != "Bearer" {
		return nil, nil, errors.New("error processing the Authorization header")
	}
	rawIDToken := splitAuthorization[1]
	tokenType = "mobile"
	return &rawIDToken, &tokenType, nil
}

func (auth *AdminAuth) verify(rawIDToken string, tokenType string) (*oidc.IDToken, error) {
	switch tokenType {
	case "mobile":
		log.Println("AdminAuth -> mobile app client token")
		return auth.appVerifier.Verify(context.Background(), rawIDToken)
	case "web":
		log.Println("AdminAuth -> web app client token")
		return auth.webAppVerifier.Verify(context.Background(), rawIDToken)
	default:
		return nil, errors.New("AdminAuth -> there is an issue with the audience")
	}
}

func (auth *AdminAuth) responseBadRequest(w http.ResponseWriter) {
	log.Println("AdminAuth -> 400 - Bad Request")

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Bad Request"))
}

func (auth *AdminAuth) responseUnauthorized(token string, w http.ResponseWriter) {
	log.Printf("AdminAuth -> 401 - Unauthorized for token %s", token)

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("Unauthorized"))
}

func (auth *AdminAuth) responseForbbiden(info string, w http.ResponseWriter) {
	log.Printf("AdminAuth -> 403 - Forbidden - %s", info)

	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Forbidden"))
}

func (auth *AdminAuth) responseInternalServerError(w http.ResponseWriter) {
	log.Println("AdminAuth -> 500 - Internal Server Error")

	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
}

func newAdminAuth(app *core.Application, oidcProvider string, appClientID string, webAppClientID string, coreTokenAuth *tokenauth.TokenAuth, authorization *casbin.Enforcer) *AdminAuth {
	provider, err := oidc.NewProvider(context.Background(), oidcProvider)
	if err != nil {
		log.Fatalln(err)
	}

	appVerifier := provider.Verifier(&oidc.Config{ClientID: appClientID})
	webAppVerifier := provider.Verifier(&oidc.Config{ClientID: webAppClientID})

	cacheUsers := &syncmap.Map{}
	lock := &sync.RWMutex{}

	auth := AdminAuth{app: app, appVerifier: appVerifier, appClientID: appClientID,
		webAppVerifier: webAppVerifier, webAppClientID: webAppClientID, coreTokenAuth: coreTokenAuth,
		cachedUsers: cacheUsers, cachedUsersLock: lock, authorization: authorization}
	return &auth
}
