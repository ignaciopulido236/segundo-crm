package crm

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

type Client struct {
	ID          string `json:"id"`
	Nombre      string `json:"nombre"`
	Apellido    string `json:"apellido"`
	Email       string `json:"email"`
	Telefono    string `json:"telefono"`
	Empresa     string `json:"empresa"`
	Cargo       string `json:"cargo"`
	Estado      string `json:"estado"`
	Origen      string `json:"origen"`
	Responsable string `json:"responsable"`
	Direccion   string `json:"direccion"`
	Ciudad      string `json:"ciudad"`
	Pais        string `json:"pais"`
	Notas       string `json:"notas"`
	Creado      string `json:"creado_en"`
	Actualizado string `json:"actualizado_en"`
}
type Meta struct {
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}
type Page struct {
	Data []Client `json:"data"`
	Meta Meta     `json:"meta"`
}
type envelope struct {
	OK     bool            `json:"ok"`
	Status int             `json:"status"`
	Data   json.RawMessage `json:"data"`
	Meta   *Meta           `json:"meta"`
	Error  struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type API struct {
	endpoint, key string
	http          *http.Client
}

var deployment = regexp.MustCompile(`^/macros/s/[A-Za-z0-9_-]+/exec$`)

func ValidateURL(raw string) error {
	u, e := url.Parse(raw)
	if e != nil || u.Scheme != "https" || u.Host != "script.google.com" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !deployment.MatchString(u.Path) {
		return errors.New("Usa una URL HTTPS de implementación: https://script.google.com/macros/s/ID/exec")
	}
	return nil
}
func redirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 5 {
		return errors.New("demasiadas redirecciones")
	}
	u := req.URL
	if u.Scheme != "https" || u.Host != "script.googleusercontent.com" || u.User != nil || u.Path != "/macros/echo" {
		return errors.New("redirección no permitida")
	}
	if req.Method != "GET" || req.Body != nil {
		return errors.New("no se reenvían credenciales en redirecciones")
	}
	return nil
}
func newAPI(endpoint, key string) *API {
	return &API{endpoint, key, &http.Client{Timeout: 45 * time.Second, CheckRedirect: redirect}}
}
func (a *API) call(action string, fields map[string]interface{}) (envelope, error) {
	var out envelope
	fields["api_key"] = a.key
	fields["action"] = action
	body, _ := json.Marshal(fields)
	req, e := http.NewRequest(http.MethodPost, a.endpoint, bytes.NewReader(body))
	if e != nil {
		return out, errors.New("Configuración inválida")
	}
	req.Header.Set("Content-Type", "application/json")
	res, e := a.http.Do(req)
	write := action == "create" || action == "update" || action == "delete"
	uncertain := func(msg string) error {
		if write {
			return errors.New(msg + ". Resultado incierto: actualiza y consulta antes de repetir la operación")
		}
		return errors.New(msg)
	}
	if e != nil {
		return out, uncertain("Problema de conexión, tiempo agotado o redirección bloqueada")
	}
	defer res.Body.Close()
	b, e := io.ReadAll(io.LimitReader(res.Body, 8*1024*1024+1))
	if e != nil || len(b) > 8*1024*1024 {
		return out, uncertain("Respuesta incompleta o demasiado grande")
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return out, uncertain(fmt.Sprintf("Error HTTP %d", res.StatusCode))
	}
	if strings.Contains(strings.ToLower(res.Header.Get("Content-Type")), "text/html") || bytes.HasPrefix(bytes.TrimSpace(b), []byte("<")) {
		return out, uncertain("Google devolvió una página HTML; revisa los permisos y la implementación")
	}
	if json.Unmarshal(b, &out) != nil {
		return out, uncertain("Respuesta JSON inválida")
	}
	// Do not echo untrusted server messages: these could contain credentials.
	if !out.OK || out.Status < 200 || out.Status >= 300 {
		switch out.Status {
		case 401, 403:
			return out, errors.New("API: acceso rechazado. Revisa la clave y permisos")
		case 409:
			return out, errors.New("API: ya existe un cliente con ese email")
		case 404:
			return out, errors.New("API: cliente no encontrado")
		case 400, 422:
			return out, errors.New("API: datos rechazados; revisa los campos")
		default:
			return out, uncertain("API: respuesta de error")
		}
	}
	expected := 200
	if action == "create" {
		expected = 201
	}
	if out.Status != expected {
		return out, uncertain("Estado JSON inesperado")
	}
	if len(out.Data) == 0 || string(out.Data) == "null" {
		return out, uncertain("Contrato JSON incompleto")
	}
	return out, nil
}

var states = map[string]bool{"nuevo": true, "contactado": true, "calificado": true, "cliente": true, "inactivo": true}
var editable = map[string]bool{"nombre": true, "apellido": true, "email": true, "telefono": true, "empresa": true, "cargo": true, "estado": true, "origen": true, "responsable": true, "direccion": true, "ciudad": true, "pais": true, "notas": true}
var emailPattern = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func validate(data map[string]string, create bool) error {
	if create && strings.TrimSpace(data["nombre"]) == "" {
		return errors.New("Nombre obligatorio")
	}
	for k, v := range data {
		if !editable[k] {
			return errors.New("Campo no editable")
		}
		max := 500
		if k == "notas" {
			max = 5000
		}
		if utf8.RuneCountInString(v) > max {
			return fmt.Errorf("%s supera el máximo de %d caracteres", k, max)
		}
		if k == "nombre" && strings.TrimSpace(v) == "" {
			return errors.New("Nombre obligatorio")
		}
		if k == "email" && v != "" {
			parsed, err := mail.ParseAddress(v)
			if err != nil || parsed.Address != v || !emailPattern.MatchString(v) {
				return errors.New("Email inválido")
			}
		}
		if k == "estado" && !states[v] {
			return errors.New("Estado inválido")
		}
	}
	return nil
}
