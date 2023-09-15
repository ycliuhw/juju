// Copyright 2017 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package apiserver

import (
	"context"
	// "encoding/json"
	// "io"
	"crypto/tls"
	"net/http"
	"time"
	// "net/url"

	"github.com/go-macaroon-bakery/macaroon-bakery/v3/bakery"
	"github.com/go-macaroon-bakery/macaroon-bakery/v3/bakery/checkers"
	// "github.com/go-macaroon-bakery/macaroon-bakery/v3/bakery/identchecker"
	"github.com/go-macaroon-bakery/macaroon-bakery/v3/httpbakery"
	"github.com/juju/errors"
	// "github.com/kr/pretty"

	"github.com/juju/juju/apiserver/apiserverhttp"
	"github.com/juju/juju/apiserver/bakeryutil"
	"github.com/juju/juju/apiserver/common/crossmodel"
	// apiservererrors "github.com/juju/juju/apiserver/errors"
	"github.com/juju/juju/core/macaroon"
	"github.com/juju/juju/state"
)

const (
	localOfferAccessLocationPath = "/offeraccess"
)

type localOfferAuthHandler struct {
	authCtx *crossmodel.AuthContext
}

func addOfferAuthHandlers(offerAuthCtxt *crossmodel.AuthContext, mux *apiserverhttp.Mux) {
	appOfferHandler := &localOfferAuthHandler{authCtx: offerAuthCtxt}
	appOfferDischargeMux := http.NewServeMux()

	discharger := httpbakery.NewDischarger(
		httpbakery.DischargerParams{
			Key:     offerAuthCtxt.OfferThirdPartyKey(),
			Checker: httpbakery.ThirdPartyCaveatCheckerFunc(appOfferHandler.checkThirdPartyCaveat),
		})
	discharger.AddMuxHandlers(appOfferDischargeMux, localOfferAccessLocationPath)

	_ = mux.AddHandler("POST", localOfferAccessLocationPath+"/discharge", appOfferDischargeMux)
	_ = mux.AddHandler("GET", localOfferAccessLocationPath+"/publickey", appOfferDischargeMux)
}

func newOfferAuthcontext(pool *state.StatePool) (*crossmodel.AuthContext, error) {
	// Create a bakery service for discharging third-party caveats for
	// local offer access authentication. This service does not persist keys;
	// its macaroons should be very short-lived.
	st, err := pool.SystemState()
	if err != nil {
		return nil, errors.Trace(err)
	}
	location := "juju model " + st.ModelUUID()
	checker := checkers.New(macaroon.MacaroonNamespace)

	// Create a bakery service for local offer access authentication. This service
	// persists keys into MongoDB in a TTL collection.
	store, err := st.NewBakeryStorage()
	if err != nil {
		return nil, errors.Trace(err)
	}
	bakeryConfig := st.NewBakeryConfig()
	key, err := bakeryConfig.GetOffersThirdPartyKey()
	if err != nil {
		return nil, errors.Trace(err)
	}
	// key.Public.Key = bakery.Key([]byte(`X8dAqHgKX++3KuWVMvrKqUvXAJd7oTqsQ1nmwTmVTjQ=`))
	locator := bakeryutil.BakeryThirdPartyLocator{PublicKey: key.Public}
	localOfferBakery := bakery.New(
		bakery.BakeryParams{
			Checker:       checker,
			RootKeyStore:  store,
			Locator:       locator,
			Key:           key,
			OpsAuthorizer: crossmodel.CrossModelAuthorizer{},
			Location:      location,
		},
	)
	localOfferBakeryKey := key
	offerBakery := &bakeryutil.ExpirableStorageBakery{
		localOfferBakery, location, localOfferBakeryKey, store, locator,
	}

	getTestBakery := func(idURL string) (*bakeryutil.ExpirableStorageBakery, error) {
		// keypair, err := bakery.GenerateKey()
		// if err != nil {
		// 	return nil, errors.Trace(err)
		// }
		// pKey, err := getPublicKey(idURL)
		// if err != nil {
		// 	return nil, errors.Trace(err)
		// }
		// logger.Criticalf("getTestBakery pKey %q", pKey.Public.String())
		// locator := bakeryutil.BakeryThirdPartyLocator{PublicKey: pKey.Public}
		// // location := idURL
		// return &bakeryutil.ExpirableStorageBakery{
		// 	Bakery: bakery.New(
		// 		bakery.BakeryParams{
		// 			Checker:      checker,
		// 			RootKeyStore: store,
		// 			Locator:      locator,
		// 			Key:          keypair,
		// 			// Key:           pKey,
		// 			OpsAuthorizer: crossmodel.CrossModelAuthorizer{},
		// 			Location:      location,
		// 		},
		// 	),
		// 	Location: location,
		// 	Key:      keypair,
		// 	// Key:     pKey,
		// 	Store:   store,
		// 	Locator: locator,
		// }, nil

		pKey, err := getPublicKey(idURL)
		if err != nil {
			return nil, errors.Trace(err)
		}
		idPK := pKey.Public
		logger.Criticalf("getTestBakery pKey %q", pKey.Public.String())
		key, err := bakeryConfig.GetExternalUsersThirdPartyKey()
		if err != nil {
			return nil, errors.Trace(err)
		}

		pkCache := bakery.NewThirdPartyStore()
		locator := httpbakery.NewThirdPartyLocator(nil, pkCache)
		pkCache.AddInfo(idURL, bakery.ThirdPartyInfo{
			PublicKey: idPK,
			Version:   3,
		})

		store, err := st.NewBakeryStorage()
		if err != nil {
			return nil, errors.Trace(err)
		}
		store = store.ExpireAfter(15 * time.Minute)
		return &bakeryutil.ExpirableStorageBakery{
			Bakery: bakery.New(
				bakery.BakeryParams{
					Checker:       checker,
					RootKeyStore:  store,
					Locator:       locator,
					Key:           key,
					OpsAuthorizer: crossmodel.CrossModelAuthorizer{},
					Location:      location,
				},
			),
			Location: location,
			Key:      key,
			Store:    store,
			Locator:  locator,
		}, nil
	}
	authCtx, err := crossmodel.NewAuthContext(
		crossmodel.GetBackend(st), key, offerBakery, getTestBakery,
	)
	if err != nil {
		return nil, err
	}
	return authCtx, nil
}

func getPublicKey(idURL string) (*bakery.KeyPair, error) {
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	thirdPartyInfo, err := httpbakery.ThirdPartyInfoForLocation(context.TODO(), &http.Client{Transport: transport}, idURL)
	logger.Criticalf("CreateMacaroonForJaaS thirdPartyInfo.Version %q, thirdPartyInfo.PublicKey.Key.String() %q", thirdPartyInfo.Version, thirdPartyInfo.PublicKey.Key.String())
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &bakery.KeyPair{Public: thirdPartyInfo.PublicKey}, nil
}

// func getOffersThirdPartyKey(st *state.State) (*bakery.KeyPair, error) {
// 	ctrlCfg, err := st.ControllerConfig()
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	loginTokenRefreshURLStr := ctrlCfg.LoginTokenRefreshURL()
// 	if loginTokenRefreshURLStr == "" {
// 		return nil, errors.NotFoundf("no login token refresh URL configured")
// 	}
// 	offerAccessEndpoint, err := url.Parse(loginTokenRefreshURLStr)
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	offerAccessEndpoint.Path = "/macaroons/publickey"
// 	resp, err := http.Get(offerAccessEndpoint.String())
// 	if err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	defer resp.Body.Close()
// 	body, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return nil, errors.Annotatef(err, "reading bad response body with status code %d", resp.StatusCode)
// 	}
// 	if resp.StatusCode != http.StatusOK {
// 		return nil, errors.Errorf(
// 			"failed to acquire macaroon public key, url %q, status %d, body %s", resp.Request.URL.String(), resp.StatusCode, body,
// 		)
// 	}
// 	var data struct {
// 		PublicKey string `json:"PublicKey"`
// 	}
// 	if err := json.Unmarshal(body, &data); err != nil {
// 		return nil, errors.Trace(err)
// 	}
// 	if data.PublicKey == "" {
// 		return nil, errors.Errorf("no public key in response body")
// 	}
// 	return &bakery.KeyPair{Public: bakery.PublicKey{bakery.Key(data.PublicKey)}}, nil
// }

func (h *localOfferAuthHandler) checkThirdPartyCaveat(stdctx context.Context, req *http.Request, cavInfo *bakery.ThirdPartyCaveatInfo, token *httpbakery.DischargeToken) (_ []checkers.Caveat, err error) {
	defer func() {
		logger.Criticalf(
			"localOfferAuthHandler.checkThirdPartyCaveat Condition %q, err %#v", string(cavInfo.Condition), err,
		)
		if token != nil {
			logger.Criticalf("localOfferAuthHandler.checkThirdPartyCaveat token.Kind %q, token.Value %q", token.Kind, string(token.Value))
		}
	}()
	logger.Debugf("check offer third party caveat %q", cavInfo.Condition)
	logger.Criticalf("localOfferAuthHandler.checkThirdPartyCaveat caveat.Condition %q", cavInfo.Condition)
	details, err := h.authCtx.CheckOfferAccessCaveat(string(cavInfo.Condition))
	if err != nil {
		return nil, errors.Trace(err)
	}

	firstPartyCaveats, err := h.authCtx.CheckLocalAccessRequest(details)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return firstPartyCaveats, nil

	// m, err := h.authCtx.CreateMacaroonForJaaS(
	// 	stdctx, details.SourceModelUUID, details.OfferUUID, details.User, details.Relation, 3,
	// )
	// if err != nil {
	// 	return nil, errors.Trace(err)
	// }
	// logger.Criticalf("check offer third party caveat %s", pretty.Sprint(m))
	// return nil, &apiservererrors.DischargeRequiredError{
	// 	Cause:          &bakery.DischargeRequiredError{Message: `https://jimm.comsys-internal.v2.staging.canonical.com/macaroons`},
	// 	Macaroon:       m,
	// 	LegacyMacaroon: m.M(),
	// }
}
