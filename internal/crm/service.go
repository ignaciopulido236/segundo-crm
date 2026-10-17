package crm

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type PublicConfig struct {
	URL        string `json:"url"`
	Configured bool   `json:"configured"`
	Problem    string `json:"problem,omitempty"`
}
type savedConfig struct {
	URL    string `json:"url"`
	Cipher []byte `json:"cipher"`
}
type Service struct {
	mu      sync.RWMutex
	api     *API
	problem string
	path    string
}

func NewService() *Service {
	dir, e := os.UserConfigDir()
	s := &Service{}
	if e != nil {
		s.problem = "No se puede acceder a la configuración"
		return s
	}
	s.path = filepath.Join(dir, "PrimerCRM", "connection.json")
	b, e := os.ReadFile(s.path)
	if os.IsNotExist(e) {
		return s
	}
	if e != nil {
		s.problem = "No se pudo leer la configuración"
		return s
	}
	var c savedConfig
	if json.Unmarshal(b, &c) != nil || ValidateURL(c.URL) != nil {
		s.problem = "Configuración dañada; vuelve a configurar"
		return s
	}
	key, e := unprotect(c.Cipher)
	if e != nil {
		s.problem = "No se pudo descifrar la clave para este usuario; vuelve a configurar"
		return s
	}
	s.api = newAPI(c.URL, string(key))
	clear(key)
	return s
}
func (s *Service) Configuration() PublicConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c := PublicConfig{Problem: s.problem}
	if s.api != nil {
		c.URL = s.api.endpoint
		c.Configured = true
	}
	return c
}
func (s *Service) Configure(endpoint, key string) error {
	if e := ValidateURL(endpoint); e != nil {
		return e
	}
	if key == "" {
		return errors.New("Introduce la clave")
	}
	api := newAPI(endpoint, key)
	if _, e := list(api, "", "", 0); e != nil {
		return e
	}
	cipher, e := protect([]byte(key))
	if e != nil {
		return errors.New("No se pudo cifrar la clave con DPAPI")
	}
	b, _ := json.Marshal(savedConfig{endpoint, cipher})
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.path == "" {
		return errors.New("Ruta de configuración no disponible")
	}
	if e = atomicSave(s.path, b); e != nil {
		return errors.New("No se pudo guardar la configuración")
	}
	s.api = api
	s.problem = ""
	return nil
}
func atomicSave(path string, b []byte) error {
	if e := os.MkdirAll(filepath.Dir(path), 0700); e != nil {
		return e
	}
	f, e := os.CreateTemp(filepath.Dir(path), ".connection-*")
	if e != nil {
		return e
	}
	name := f.Name()
	defer os.Remove(name)
	if e = f.Chmod(0600); e == nil {
		_, e = f.Write(b)
	}
	if e == nil {
		e = f.Sync()
	}
	ce := f.Close()
	if e == nil {
		e = ce
	}
	if e != nil {
		return e
	}
	return replaceFile(name, path)
}
func (s *Service) connection() (*API, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.api == nil {
		return nil, errors.New("Configura la conexión")
	}
	return s.api, nil
}
func list(a *API, q, state string, offset int) (Page, error) {
	var p Page
	if offset < 0 || offset%50 != 0 || state != "" && !states[state] {
		return p, errors.New("Paginación o estado inválido")
	}
	r, e := a.call("list", map[string]interface{}{"q": q, "estado": state, "limit": 50, "offset": offset})
	if e != nil {
		return p, e
	}
	if json.Unmarshal(r.Data, &p.Data) != nil || p.Data == nil || r.Meta == nil || r.Meta.Limit != 50 || r.Meta.Offset != offset || r.Meta.Total < 0 {
		return p, errors.New("Contrato list inválido")
	}
	p.Meta = *r.Meta
	return p, nil
}
func (s *Service) List(q, state string, offset int) (Page, error) {
	a, e := s.connection()
	if e != nil {
		return Page{}, e
	}
	return list(a, q, state, offset)
}
func (s *Service) Get(id string) (Client, error) {
	var c Client
	if id == "" {
		return c, errors.New("ID obligatorio")
	}
	a, e := s.connection()
	if e != nil {
		return c, e
	}
	r, e := a.call("get", map[string]interface{}{"id": id})
	if e != nil {
		return c, e
	}
	if json.Unmarshal(r.Data, &c) != nil || c.ID != id {
		return c, errors.New("Contrato get inválido")
	}
	return c, nil
}
func (s *Service) Write(action, id string, data map[string]string) (Client, error) {
	var c Client
	if action != "create" && action != "update" {
		return c, errors.New("Acción inválida")
	}
	if action == "update" && id == "" {
		return c, errors.New("ID obligatorio")
	}
	if e := validate(data, action == "create"); e != nil {
		return c, e
	}
	a, e := s.connection()
	if e != nil {
		return c, e
	}
	r, e := a.call(action, map[string]interface{}{"id": id, "data": data})
	if e != nil {
		return c, e
	}
	if json.Unmarshal(r.Data, &c) != nil || c.ID == "" || (action == "update" && c.ID != id) {
		return c, errors.New("Respuesta de escritura inválida. Resultado incierto: actualiza antes de repetir")
	}
	return c, nil
}
func (s *Service) Delete(id string) error {
	if id == "" {
		return errors.New("ID obligatorio")
	}
	a, e := s.connection()
	if e != nil {
		return e
	}
	r, e := a.call("delete", map[string]interface{}{"id": id})
	if e != nil {
		return e
	}
	var d struct {
		ID      string `json:"id"`
		Deleted bool   `json:"eliminado"`
	}
	if json.Unmarshal(r.Data, &d) != nil || d.ID != id || !d.Deleted {
		return errors.New("Respuesta de eliminación inválida. Resultado incierto: actualiza antes de repetir")
	}
	return nil
}
