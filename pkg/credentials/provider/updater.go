package provider

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type getCredentialsFunc func(ctx context.Context) (*Credentials, error)

type Updater struct {
	expiryWindow  time.Duration
	refreshPeriod time.Duration

	// for fix below case:
	// * both auth.Signer and credential.Credential are not concurrent safe
	expiryWindowForRefreshLoop time.Duration

	getCredentials func(ctx context.Context) (*Credentials, error)

	cred        *Credentials
	lockForCred sync.RWMutex

	Logger  Logger
	nowFunc func() time.Time
}

type UpdaterOptions struct {
	ExpiryWindow  time.Duration
	RefreshPeriod time.Duration
	Logger        Logger
}

func NewUpdater(getter getCredentialsFunc, opts UpdaterOptions) *Updater {
	u := &Updater{
		expiryWindow:               opts.ExpiryWindow,
		refreshPeriod:              opts.RefreshPeriod,
		expiryWindowForRefreshLoop: opts.RefreshPeriod + opts.RefreshPeriod/2,
		getCredentials:             getter,
		cred:                       nil,
		lockForCred:                sync.RWMutex{},
		Logger:                     opts.Logger,
		nowFunc:                    time.Now,
	}
	return u
}

func (u *Updater) Start(ctx context.Context) {
	if u.refreshPeriod <= 0 {
		return
	}

	go u.startRefreshLoop(ctx)
}

func (u *Updater) startRefreshLoop(ctx context.Context) {
	ticket := time.NewTicker(u.refreshPeriod)
	defer ticket.Stop()

	u.refreshCredForLoop(ctx)

loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-ticket.C:
			u.refreshCredForLoop(ctx)
		}
	}
}

func (u *Updater) Credentials(ctx context.Context) (*Credentials, error) {
	if u.Expired() {
		if err := u.refreshCred(ctx); err != nil {
			return nil, err
		}
	}

	cred := u.getCred().DeepCopy()
	return &cred, nil
}

func (u *Updater) refreshCredForLoop(ctx context.Context) {
	exp := u.expiration()

	if !u.expired(u.expiryWindowForRefreshLoop) {
		return
	}

	u.logger().Debug(fmt.Sprintf("start refresh credentials, current expiration: %s",
		exp.Format("2006-01-02T15:04:05Z")))

	maxRetry := 5
	for i := 0; i < maxRetry; i++ {
		err := u.refreshCred(ctx)
		if err == nil {
			return
		}
		if _, ok := err.(*NotEnableError); ok {
			return
		}
		if i < maxRetry-1 {
			time.Sleep(time.Second * time.Duration(i))
		}
	}
}

func (u *Updater) refreshCred(ctx context.Context) error {
	cred, err := u.getCredentials(ctx)
	if err != nil {
		if _, ok := err.(*NotEnableError); ok {
			return err
		}
		u.logger().Error(err, fmt.Sprintf("refresh credentials failed: %s", err))
		return err
	}
	u.logger().Debug(fmt.Sprintf("refreshed credentials, expiration: %s",
		cred.Expiration.Format("2006-01-02T15:04:05Z")))

	u.setCred(*cred)
	return nil
}

func (u *Updater) setCred(cred Credentials) {
	u.lockForCred.Lock()
	defer u.lockForCred.Unlock()

	cred = cred.DeepCopy()
	cred.Expiration = cred.Expiration.Round(0)
	if u.expiryWindow > 0 {
		cred.Expiration = cred.Expiration.Add(-u.expiryWindow)
	}
	u.cred = &cred
}

func (u *Updater) getCred() *Credentials {
	u.lockForCred.RLock()
	defer u.lockForCred.RUnlock()

	return u.cred
}

func (u *Updater) Expired() bool {
	return u.expired(0)
}

func (u *Updater) expired(expiryDelta time.Duration) bool {
	exp := u.expiration()

	return exp.Add(-expiryDelta).Before(u.now())
}

func (u *Updater) expiration() time.Time {
	cred := u.getCred()

	if cred == nil {
		return time.Time{}
	}

	return cred.Expiration.Round(0)
}

func (u *Updater) now() time.Time {
	if u.nowFunc == nil {
		return time.Now()
	}
	return u.nowFunc()
}

func (u *Updater) logger() Logger {
	if u.Logger != nil {
		return u.Logger
	}
	return defaultLog
}
