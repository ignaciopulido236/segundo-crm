//go:build windows || bindings

package main

import (
	"context"
	"embed"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"log"
	"segundo-crm/internal/crm"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := &App{service: crm.NewService()}
	err := wails.Run(&options.App{Title: "PrimerCRM", Width: 1280, Height: 820, MinWidth: 960, MinHeight: 640, AssetServer: &assetserver.Options{Assets: assets}, Bind: []interface{}{app}, OnBeforeClose: func(ctx context.Context) bool {
		if app.pending.Load() {
			_, _ = runtime.MessageDialog(ctx, runtime.MessageDialogOptions{Type: runtime.InfoDialog, Title: "Operación pendiente", Message: "Espera a que termine la operación antes de cerrar."})
			return true
		}
		if !app.dirty.Load() {
			return false
		}
		answer, err := runtime.MessageDialog(ctx, runtime.MessageDialogOptions{Type: runtime.QuestionDialog, Title: "Cambios sin guardar", Message: "¿Descartar los cambios y cerrar PrimerCRM?", Buttons: []string{"Yes", "No"}, DefaultButton: "No", CancelButton: "No"})
		return err != nil || answer != "Yes"
	}})
	if err != nil {
		log.Print("No se pudo iniciar PrimerCRM")
	}
}
