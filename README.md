# go-tdameritrade
go client for the tdameritrade api, forked from [Zachray Rice's library](https://github.com/zricethezav/go-tdameritrade)

[![Documentation](https://godoc.org/github.com/JonCooperWorks/go-tdameritrade?status.svg)](https://godoc.org/github.com/JonCooperWorks/go-tdameritrade)


```import "github.com/JonCooperWorks/go-tdameritrade"```

go-tdameritrade handles all interaction with the [TD Ameritrade REST API](https://developer.tdameritrade.com/apis).
See the TD Ameritrade [developer site](https://developer.tdameritrade.com/) to learn how their APIs work.
This is a very thin wrapper and does not perform any validation.


## Authentication with TD Ameritrade
There is an example of using OAuth2 to authenticate a user and use the services on the TD Ameritrade API in [examples/webauth/webauth.go](./examples/webauth/webauth.go).
Authentication is handled by the ```Authenticator``` struct and its methods ```StartOAuth2Flow``` and ```FinishOAuth2Flow```.
You can get an authenticated ```tdameritrade.Client``` from an authenticated request with the ```AuthenticatedClient``` method, and use that to interact with the TD API.
See [auth.go](./auth.go).

```golang
// Authenticator is a helper for TD Ameritrade's authentication.
// It authenticates users and validates the state returned from TD Ameritrade to protect users from CSRF attacks.
// It's recommended to use NewAuthenticator instead of creating this struct directly because TD Ameritrade requires Client IDs to be in the form clientid@AMER.OAUTHAP.
// This is not immediately obvious from the documentation.
// See https://developer.tdameritrade.com/content/authentication-faq
type Authenticator struct {
	Store  PersistentStore
	OAuth2 oauth2.Config
}
```

The library handles state generation and the OAuth2 flow.
Users simply implement the ```PersistentStore``` interface (see [auth.go](./auth.go)) and tell it how to store and retrieve OAuth2 state and an ```oauth2.Token``` with the logged in user's credentials.

```golang
// PersistentStore is meant to persist data from TD Ameritrade that is needed between requests.
// Implementations must return the same value they set for a user in StoreState in GetState, or the login process will fail.
// It is meant to allow credentials to be stored in cookies, JWTs and anything else you can think of.
type PersistentStore interface {
	StoreToken(token *oauth2.Token, w http.ResponseWriter, req *http.Request) error
	GetToken(req *http.Request) (*oauth2.Token, error)
	StoreState(state string, w http.ResponseWriter, req *http.Request) error
	GetState(*http.Request) (string, error)
}
```

## Interacting with the TD Ameritrade API
The library is centered around the ```tdameritrade.Client```.
It allows access to all services exposed by the TD Ameritrade REST API.
More information about each service can be found on TD Ameritrade's [developer website](https://developer.tdameritrade.com/apis).

```golang
// A Client manages communication with the TD-Ameritrade API.
type Client struct {
	client *http.Client // HTTP client used to communicate with the API.

	// Base URL for API requests. Defaults to the public TD-Ameritrade API, but can be
	// set to any endpoint. This allows for more manageable testing.
	BaseURL *url.URL

	// services used for talking to different parts of the tdameritrade api
	PriceHistory       *PriceHistoryService
	Account            *AccountsService
	MarketHours        *MarketHoursService
	Quotes             *QuotesService
	Instrument         *InstrumentService
	Chains             *ChainsService
	Mover              *MoverService
	TransactionHistory *TransactionHistoryService
	User               *UserService
	Watchlist          *WatchlistService
}
```

You get a ```tdameritrade.Client``` from the ```FinishOAuth2``` or ```AuthenticatedClient``` method on the ```tdameritrade.Authenticator``` struct.

## Streaming
TD Ameritrade provides a [websockets API](https://developer.tdameritrade.com/content/streaming-data) that allows for streaming data.
`go-tdameritrade` provides a [streaming client](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#StreamingClient) for authenticating with TD Ameritrade's socket API.
To create an instance of an authenticated `tdameritrade.StreamingClient`, use the [`tdameritrade.NewAuthenticatedStreamingClient`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#NewAuthenticatedStreamingClient) method.
You'll need an authenticated [`tdameritrade.Client`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#Client) and the account ID you intend to listen on.
To manually create an authenticated socket, use [`tdameritrade.NewUnauthenticatedStreamingClient`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#NewUnauthenticatedStreamingClient) to create a socket and send a JSON encoded authentication message.
A convenience method, [`tdameritrade.NewStreamAuthCommand`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#NewStreamAuthCommand), will create one of these given a [UserPrincipal](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#UserService.GetUserPrincipals) and an account ID.


TD's Streaming API accepts commands in a format described in their [documentation](https://developer.tdameritrade.com/content/streaming-data#_Toc504640563).
`go-tdameritrade` provides struct wrappers over the command types.

```golang
type Command struct {
	Requests []StreamRequest `json:"requests"`
}

type StreamRequest struct {
	Service    string       `json:"service"`
	Requestid  string       `json:"requestid"`
	Command    string       `json:"command"`
	Account    string       `json:"account"`
	Source     string       `json:"source"`
	Parameters StreamParams `json:"parameters"`
}

type StreamParams struct {
	Keys   string `json:"keys"`
	Fields string `json:"fields"`
}
```

[`Command`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#Command)s can be sent to TD Ameritrade with the [`SendCommand`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#StreamingClient.SendCommand) convenience method.

```golang
streamingClient.SendCommand(tdameritrade.Command{
	Requests: []tdameritrade.StreamRequest{
		{
			Service:   "QUOTE",
			Requestid: "2",
			Command:   "SUBS",
			Account:   userPrincipals.Accounts[0].AccountID,
			Source:    userPrincipals.StreamerInfo.AppID,
			Parameters: tdameritrade.StreamParams{
				Keys:   "AAPL",
				Fields: "0,1,2,3,4,5,6,7,8",
			},
		},
	},
})
```

Streaming support is very basic.
`go-tdameritrade` can only receive `[]byte` payloads from a TD Ameritrade websocket with the [`ReceiveText`](https://pkg.go.dev/github.com/joncooperworks/go-tdameritrade#StreamingClient.ReceiveText) method.
You can find an example [here](examples/streaming/streaming.go#L112).



## Examples

More examples are in the [examples](./examples) directory.

#### Configuring the Authenticator from an environment variable

```golang
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
		RedirectURL: "https://localhost:8080/callback",
	},
)
```

#### Authenticating a user with OAuth2
```golang
type TDHandlers struct {
	authenticator *tdameritrade.Authenticator
}

func (h *TDHandlers) Authenticate(w http.ResponseWriter, req *http.Request) {
	redirectURL, err := h.authenticator.StartOAuth2Flow(w, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	http.Redirect(w, req, redirectURL, http.StatusTemporaryRedirect)
}

func (h *TDHandlers) Callback(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	_, err := h.authenticator.FinishOAuth2Flow(ctx, w, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	http.Redirect(w, req, "/quote?ticker=SPY", http.StatusTemporaryRedirect)
}
```

#### Looking up a stock quote using the API.
```golang
type TDHandlers struct {
	authenticator *tdameritrade.Authenticator
}

func (h *TDHandlers) Quote(w http.ResponseWriter, req *http.Request) {
	ctx := context.Background()
	client, err := h.authenticator.AuthenticatedClient(ctx, req)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	ticker, ok := req.URL.Query()["ticker"]
	if !ok || len(ticker) == 0 {
		w.Write([]byte("ticker is required"))
		return
	}

	quote, _, err := client.Quotes.GetQuotes(ctx, ticker[0])
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	body, err := json.Marshal(quote)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	w.Write(body)

}
```


Use at your own risk.
