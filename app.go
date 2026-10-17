package main

import (
	"segundo-crm/internal/crm"
	"sync/atomic"
)

// App is the complete frontend boundary. No credential getter is exported.
type App struct {
	service *crm.Service
	dirty   atomic.Bool
	pending atomic.Bool
}

func (a *App) Configuration() crm.PublicConfig { return a.service.Configuration() }
func (a *App) Configure(url, key string) error { return a.service.Configure(url, key) }
func (a *App) List(q, estado string, offset int) (crm.Page, error) {
	return a.service.List(q, estado, offset)
}
func (a *App) Get(id string) (crm.Client, error) { return a.service.Get(id) }
func (a *App) Create(data map[string]string) (crm.Client, error) {
	return a.service.Write("create", "", data)
}
func (a *App) Update(id string, data map[string]string) (crm.Client, error) {
	return a.service.Write("update", id, data)
}
func (a *App) Delete(id string) error { return a.service.Delete(id) }

// SetSessionState lets native window close protect unsaved input.
func (a *App) SetSessionState(dirty, pending bool) { a.dirty.Store(dirty); a.pending.Store(pending) }
