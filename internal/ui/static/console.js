// Console — executes GraphQL queries via fetch()
(function() {
    "use strict";

    window.executeQuery = function() {
        var schemaSelect = document.getElementById("schema-select");
        var queryInput = document.getElementById("query-input");
        var resultOutput = document.getElementById("result-output");

        var schemaName = schemaSelect.value;
        if (!schemaName) {
            resultOutput.textContent = "Fehler: Kein Schema ausgewählt.";
            resultOutput.className = "result-box message-error";
            return;
        }

        var query = queryInput.value.trim();
        if (!query) {
            resultOutput.textContent = "Fehler: Kein Query eingegeben.";
            resultOutput.className = "result-box message-error";
            return;
        }

        var headers = {"Content-Type": "application/json"};
        var apiKey = localStorage.getItem("stratum-api-key");
        if (apiKey) {
            headers["X-API-Key"] = apiKey;
        }

        resultOutput.textContent = "Wird ausgeführt...";
        resultOutput.className = "result-box";

        fetch("/graphql/" + encodeURIComponent(schemaName), {
            method: "POST",
            headers: headers,
            body: JSON.stringify({query: query})
        })
        .then(function(resp) {
            return resp.json().then(function(data) {
                return {status: resp.status, data: data};
            });
        })
        .then(function(result) {
            resultOutput.textContent = JSON.stringify(result.data, null, 2);
            if (result.data.errors || result.status !== 200) {
                resultOutput.className = "result-box message-error";
            } else {
                resultOutput.className = "result-box message-success";
            }
        })
        .catch(function(err) {
            resultOutput.textContent = "Fehler: " + err.message;
            resultOutput.className = "result-box message-error";
        });
    };
})();
