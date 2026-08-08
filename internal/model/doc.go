// Package model is the root Bubble Tea model of postgo: the single
// tea.Model the program runs, plus everything the root needs to respond
// to messages.
//
// # Layering
//
// Two kinds of code live here, deliberately:
//
//  1. The tea core — the framework surface: Init/Update/View, the tea.Msg
//     types, the pane focus machinery, and the key routing. Files:
//     model.go (state + Update router), view.go (View + rendering),
//     focus.go (pane enum + enter()/cycleFocus), keys.go (binding table).
//
//  2. The handlers — one file per operation that Update delegates to:
//     actions.go (action registry + palette overlay), send.go,
//     collection.go (save/rename/create), confirm.go (delete flow),
//     env.go (environment manager), namer.go, curl.go, session.go.
//     They are not framework code; they live here because they mutate
//     widget state (sidebar, editor, urlbar, overlays), which only the
//     model may touch.
//
// # Rule
//
// Pure logic does not belong in this package. The application layer
// (internal/app) owns operations that need no widgets: the send pipeline,
// collection mutations, curl formatting, and the shared vars helpers.
// If a function here touches no widget, overlay, or focus state, it
// belongs in app/collection instead.
//
// See [[Architecture Overview]] and [[Root Model and Focus Routing]] in
// the project vault for the full picture.
package model
