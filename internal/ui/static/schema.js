(function() {
    "use strict";

    var CM = window.CodeMirror6;
    var GQL = window.GraphQLWeb;

    var editorView = new CM.EditorView({
        state: CM.EditorState.create({
            doc: "",
            extensions: [CM.basicSetup, CM.graphql()]
        }),
        parent: document.getElementById("editor")
    });

    function getSDL() {
        return editorView.state.doc.toString();
    }

    function setSDL(text) {
        editorView.dispatch({
            changes: {from: 0, to: editorView.state.doc.length, insert: text}
        });
    }

    function getSchemaName() {
        return document.getElementById("schema-name").value.trim();
    }

    function getAPIKey() {
        return localStorage.getItem("stratum-api-key") || "";
    }

    function showMessage(text, isError) {
        var area = document.getElementById("message-area");
        area.innerHTML = '<div class="message ' + (isError ? 'message-error' : 'message-success') + '">' +
            text.replace(/</g, '&lt;') + '</div>';
    }

    function clearDiagnostics() {
        editorView.dispatch(CM.setDiagnostics(editorView.state, []));
    }

    window.loadSchema = function(name) {
        var entries = document.querySelectorAll(".schema-entry");
        for (var i = 0; i < entries.length; i++) {
            if (entries[i].getAttribute("data-name") === name) {
                document.getElementById("schema-name").value = name;
                setSDL(entries[i].getAttribute("data-sdl"));
                clearDiagnostics();
                break;
            }
        }
    };

    window.formatSDL = function() {
        var sdl = getSDL();
        if (!sdl.trim()) return;
        try {
            var ast = GQL.parse(sdl);
            var formatted = GQL.print(ast);
            setSDL(formatted);
            clearDiagnostics();
        } catch (e) {
            showMessage("Syntaxfehler: " + e.message, true);
        }
    };

    window.lintSDL = function() {
        var name = getSchemaName();
        if (!name) {
            showMessage("Bitte Schema-Name eingeben", true);
            return;
        }
        var sdl = getSDL();
        if (!sdl.trim()) return;

        var apiKey = getAPIKey();
        var headers = {"Content-Type": "application/json"};
        if (apiKey) headers["X-API-Key"] = apiKey;

        fetch("/api/v1/schemas/" + encodeURIComponent(name) + "?preview=true", {
            method: "POST",
            headers: headers,
            body: JSON.stringify({sdl: sdl})
        })
        .then(function(resp) { return resp.json().then(function(data) { return {status: resp.status, data: data}; }); })
        .then(function(result) {
            if (result.status === 200) {
                clearDiagnostics();
                showMessage("SDL ist gültig", false);
                return;
            }
            if (result.status === 422 && result.data.details) {
                var diagnostics = [];
                result.data.details.forEach(function(d) {
                    var line = (d.line || 1) - 1;
                    var col = (d.column || 1) - 1;
                    var docLine = editorView.state.doc.line(Math.min(line + 1, editorView.state.doc.lines));
                    var from = docLine.from + Math.min(col, docLine.length);
                    var to = Math.min(from + 1, docLine.to);
                    diagnostics.push({from: from, to: to, severity: "error", message: d.message || "Validation error"});
                });
                editorView.dispatch(CM.setDiagnostics(editorView.state, diagnostics));
                showMessage("Validierungsfehler gefunden", true);
            } else {
                showMessage(result.data.message || "Fehler bei der Validierung", true);
            }
        })
        .catch(function(err) {
            showMessage("Netzwerkfehler: " + err.message, true);
        });
    };

    window.uploadSchema = function() {
        var name = getSchemaName();
        if (!name) {
            showMessage("Bitte Schema-Name eingeben", true);
            return;
        }
        var sdl = getSDL();
        if (!sdl.trim()) {
            showMessage("Bitte SDL eingeben", true);
            return;
        }

        var apiKey = getAPIKey();
        if (!apiKey) {
            showMessage("Kein API-Key gesetzt — bitte in der Sidebar eingeben", true);
            return;
        }

        fetch("/api/v1/schemas/" + encodeURIComponent(name), {
            method: "POST",
            headers: {"Content-Type": "application/json", "X-API-Key": apiKey},
            body: JSON.stringify({sdl: sdl})
        })
        .then(function(resp) { return resp.json().then(function(data) { return {status: resp.status, data: data}; }); })
        .then(function(result) {
            if (result.status === 200) {
                showMessage("Schema '" + result.data.name + "' erfolgreich hochgeladen (Version " + result.data.version + ")", false);
                htmx.ajax('GET', '/ui/schema/list', {target: '#schema-list-content', swap: 'innerHTML'});
            } else {
                showMessage(result.data.message || "Fehler beim Upload", true);
            }
        })
        .catch(function(err) {
            showMessage("Netzwerkfehler: " + err.message, true);
        });
    };
})();
