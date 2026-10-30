// Package googletest é um Google falso (httptest) usado pelos testes de OAuth, Drive e Slides.
package googletest

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"golang.org/x/oauth2"
)

// Fake responde às rotas de OAuth, Drive e Slides usadas pelo gerador.
type Fake struct {
	mu sync.Mutex

	Email string

	Copias        int
	Revogados     int
	UltimoNome    string
	UltimaPasta   string
	Replaces      [][2]string
	CreateImages  []string
	Escalas       []float64
	Larguras      []float64
	ReplaceImages int
	CreateTables  int
	CreateShapes  int
	Deletes       int
	Batchs        int
	Chamadas      []string
	Pastas        int
	Imagens       []string

	Presentation any // JSON da apresentação devolvida no GET

	Servidor *httptest.Server
}

// Novo cria e inicia o servidor falso.
func Novo() *Fake {
	f := &Fake{Email: "ana@rpe.tech", Presentation: map[string]any{
		"presentationId": "copia-1",
		"slides": []any{
			map[string]any{
				"objectId": "p1",
				"pageElements": []any{
					map[string]any{
						"objectId": "t1",
						"shape": map[string]any{
							"text": map[string]any{
								"textElements": []any{
									map[string]any{"textRun": map[string]any{"content": "{{HPAY_T3_E_TOT}} {{PERIODO_INICIO}}"}},
								},
							},
						},
					},
				},
			},
		},
	}}
	f.Servidor = httptest.NewServer(http.HandlerFunc(f.tratar))
	return f
}

// URL do servidor.
func (f *Fake) URL() string { return f.Servidor.URL }

// Fechar encerra o servidor.
func (f *Fake) Fechar() { f.Servidor.Close() }

// Endpoint OAuth apontando para este servidor.
func (f *Fake) Endpoint() oauth2.Endpoint {
	return oauth2.Endpoint{AuthURL: f.URL() + "/auth", TokenURL: f.URL() + "/token"}
}

// DriveURL é o endpoint da API Drive (com barra no fim).
func (f *Fake) DriveURL() string { return f.URL() + "/drive/v3/" }

// SlideURL é o endpoint da API Slides (com barra no fim).
func (f *Fake) SlideURL() string { return f.URL() + "/slides/v1/" }

func (f *Fake) tratar(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	f.Chamadas = append(f.Chamadas, r.Method+" "+r.URL.Path)
	f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.URL.Path == "/auth":
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	case r.URL.Path == "/token":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token": "access-teste", "refresh_token": "refresh-teste", "token_type": "Bearer", "expires_in": 3600,
		})
	case r.URL.Path == "/revoke":
		f.mu.Lock()
		f.Revogados++
		f.mu.Unlock()
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"revoked":true}`))
	case strings.Contains(r.URL.Path, "/about"):
		_ = json.NewEncoder(w).Encode(map[string]any{"user": map[string]any{"emailAddress": f.Email, "displayName": "Ana"}})
	case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/files/") && strings.HasSuffix(r.URL.Path, "/copy"):
		f.mu.Lock()
		f.Copias++
		var corpo struct {
			Name    string   `json:"name"`
			Parents []string `json:"parents"`
		}
		_ = json.NewDecoder(r.Body).Decode(&corpo)
		f.UltimoNome = corpo.Name
		if len(corpo.Parents) > 0 {
			f.UltimaPasta = corpo.Parents[0]
		}
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "copia-1", "name": corpo.Name})
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/thumbnail"):
		_ = json.NewEncoder(w).Encode(map[string]any{"contentUrl": "https://example.com/slide-thumb.png", "width": 1600, "height": 720})
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/presentations/"):
		_ = json.NewEncoder(w).Encode(f.Presentation)
	case r.Method == http.MethodPost && strings.Contains(r.URL.Path, ":batchUpdate"):
		raw, _ := io.ReadAll(r.Body)
		var corpo struct {
			Requests []struct {
				ReplaceAllText *struct {
					ContainsText *struct {
						Text string `json:"text"`
					} `json:"containsText"`
					ReplaceText string `json:"replaceText"`
				} `json:"replaceAllText"`
				CreateImage *struct {
					ObjectId          string `json:"objectId"`
					ElementProperties *struct {
						Size *struct {
							Width *struct {
								Magnitude float64 `json:"magnitude"`
							} `json:"width"`
						} `json:"size"`
						Transform *struct {
							ScaleX float64 `json:"scaleX"`
						} `json:"transform"`
					} `json:"elementProperties"`
				} `json:"createImage"`
				ReplaceAllShapesWithImage *struct {
					ImageUrl string `json:"imageUrl"`
				} `json:"replaceAllShapesWithImage"`
				CreateTable *struct {
					ObjectId string `json:"objectId"`
				} `json:"createTable"`
				CreateShape *struct {
					ObjectId string `json:"objectId"`
				} `json:"createShape"`
				DeleteObject *struct {
					ObjectId string `json:"objectId"`
				} `json:"deleteObject"`
			} `json:"requests"`
		}
		_ = json.Unmarshal(raw, &corpo)
		f.mu.Lock()
		f.Batchs++
		for _, req := range corpo.Requests {
			if req.ReplaceAllText != nil && req.ReplaceAllText.ContainsText != nil {
				f.Replaces = append(f.Replaces, [2]string{req.ReplaceAllText.ContainsText.Text, req.ReplaceAllText.ReplaceText})
			}
			if req.CreateImage != nil {
				f.CreateImages = append(f.CreateImages, req.CreateImage.ObjectId)
				if p := req.CreateImage.ElementProperties; p != nil {
					if p.Transform != nil {
						f.Escalas = append(f.Escalas, p.Transform.ScaleX)
					}
					if p.Size != nil && p.Size.Width != nil {
						f.Larguras = append(f.Larguras, p.Size.Width.Magnitude)
					}
				}
			}
			if req.ReplaceAllShapesWithImage != nil {
				f.ReplaceImages++
			}
			if req.CreateTable != nil {
				f.CreateTables++
			}
			if req.CreateShape != nil {
				f.CreateShapes++
			}
			if req.DeleteObject != nil {
				f.Deletes++
			}
		}
		f.mu.Unlock()
		_ = json.NewEncoder(w).Encode(map[string]any{"presentationId": "copia-1"})
	case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/files") && !strings.Contains(r.URL.Path, "/copy"):
		raw, _ := io.ReadAll(r.Body)
		meta := metaDoUpload(raw)
		f.mu.Lock()
		if strings.Contains(meta.MimeType, "folder") {
			f.Pastas++
		}
		if meta.MimeType == "image/png" || strings.HasSuffix(strings.ToLower(meta.Name), ".png") {
			f.Imagens = append(f.Imagens, meta.Name)
		}
		f.mu.Unlock()
		id := "img-1"
		if strings.Contains(meta.MimeType, "folder") {
			id = "pasta-img-1"
		}
		if strings.Contains(r.URL.RawQuery, "uploadType=resumable") {
			w.Header().Set("Location", f.URL()+"/upload/drive/v3/files/"+id)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": id, "name": meta.Name, "mimeType": meta.MimeType,
			"thumbnailLink": "https://example.com/thumb.png", "webContentLink": "https://example.com/img.png",
		})
	case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/files"):
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "img-1", "mimeType": "image/png",
			"thumbnailLink": "https://example.com/thumb.png", "webContentLink": "https://example.com/img.png",
		})
	case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/files/"):
		id := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		if id == "" {
			id = "img-1"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": id, "name": "Métricas teste", "parents": []string{"pasta-destino"},
			"thumbnailLink": "https://example.com/thumb.png", "webContentLink": "https://example.com/img.png",
		})
	case strings.Contains(r.URL.Path, "/permissions"):
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "perm-1"})
	case r.Method == http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"message":"rota desconhecida ` + r.URL.Path + `"}}`))
	}
}

func metaDoUpload(raw []byte) struct {
	Name     string `json:"name"`
	MimeType string `json:"mimeType"`
} {
	var meta struct {
		Name     string `json:"name"`
		MimeType string `json:"mimeType"`
	}
	if json.Unmarshal(raw, &meta) == nil && (meta.Name != "" || meta.MimeType != "") {
		return meta
	}
	s := string(raw)
	if i := strings.Index(s, `{"`); i >= 0 {
		if j := strings.Index(s[i:], "}"); j >= 0 {
			_ = json.Unmarshal([]byte(s[i:i+j+1]), &meta)
		}
	}
	return meta
}
