package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/joncooperworks/go-tdameritrade"
	"golang.org/x/oauth2"
)

type HTTPHeaderStore struct{}

func (s *HTTPHeaderStore) StoreToken(token *oauth2.Token, w http.ResponseWriter, req *http.Request) error {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	http.SetCookie(
		w,
		&http.Cookie{
			Name:    "refreshToken",
			Value:   token.RefreshToken,
			Expires: token.Expiry,
		},
	)
	http.SetCookie(
		w,
		&http.Cookie{
			Name:    "accessToken",
			Value:   token.AccessToken,
			Expires: token.Expiry,
		},
	)
	return nil
}

func (s HTTPHeaderStore) GetToken(req *http.Request) (*oauth2.Token, error) {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	refreshToken, err := req.Cookie("refreshToken")
	if err != nil {
		return nil, err
	}

	accessToken, err := req.Cookie("accessToken")
	if err != nil {
		return nil, err
	}

	return &oauth2.Token{
		AccessToken:  accessToken.Value,
		RefreshToken: refreshToken.Value,
		Expiry:       refreshToken.Expires,
	}, nil
}

func (s HTTPHeaderStore) StoreState(state string, w http.ResponseWriter, req *http.Request) error {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	http.SetCookie(
		w,
		&http.Cookie{
			Name:  "state",
			Value: state,
		},
	)
	return nil
}

func (s HTTPHeaderStore) GetState(req *http.Request) (string, error) {
	// DO NOT DO THIS IN A PRODUCTION ENVIRONMENT!
	// This is just an example.
	// Used signed cookies like those provided by https://github.com/gorilla/securecookie
	cookie, err := req.Cookie("state")
	if err != nil {
		return "", err
	}

	return cookie.Value, nil
}

type TDHandlers struct {
	authenticator *tdameritrade.Authenticator
}

func (h *TDHandlers) Authenticate(w http.ResponseWriter, req *http.Request) {
	redirectURL, err := h.authenticator.StartOAuth2Flow(w, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	http.Redirect(w, req, redirectURL, http.StatusTemporaryRedirect)
}

func (h *TDHandlers) Callback(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	c, err := h.authenticator.FinishOAuth2Flow(ctx, w, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	userPrincipals, resp, err := c.User.GetUserPrincipals(ctx, "streamerSubscriptionKeys", "streamerConnectionInfo")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("User Principals: [Status: %d] %+v", resp.StatusCode, *userPrincipals)

	go func() {
		streamingClient, err := tdameritrade.AuthenticatedStreamingClient(ctx, c, userPrincipals.Accounts[0].AccountID)
		if err != nil {
			log.Fatal(err)
		}
		defer streamingClient.Close()
		messages, errors := streamingClient.ReceiveText()
		for {
			select {
			case message := <-messages:
				log.Printf("message: %s", message)

			case err := <-errors:
				log.Printf("error: %v", err)
			default:
				continue
			}
		}
	}()

}

func main() {
	clientID := os.Getenv("TDAMERITRADE_CLIENT_ID")
	if clientID == "" {
		log.Fatal("Unauthorized: No client ID present")
	}

	authenticator := tdameritrade.NewAuthenticator(
		&HTTPHeaderStore{},
		oauth2.Config{
			ClientID: clientID,
			Endpoint: oauth2.Endpoint{
				TokenURL: "https://api.tdameritrade.com/v1/oauth2/token",
				AuthURL:  "https://auth.tdameritrade.com/auth",
			},
			RedirectURL: "https://127.0.0.1/callback",
		},
	)
	handlers := &TDHandlers{authenticator: authenticator}
	http.HandleFunc("/authenticate", handlers.Authenticate)
	http.HandleFunc("/callback", handlers.Callback)
	log.Fatal(http.ListenAndServe(":8000", nil))
}
