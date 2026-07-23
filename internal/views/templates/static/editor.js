const monacoBase = "https://cdn.jsdelivr.net/npm/monaco-editor@0.55.1/min/vs";

require.config({
  paths: {
    vs: monacoBase,
  },
});

const defaultOpenAPI = `openapi: 3.0.3

info:
  title: Playground API
  version: 1.0.0

paths:
  /users/{id}:
    get:
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string

        - name: debug
          in: query
          required: false
          schema:
            type: boolean

      responses:
        "200":
          description: User returned successfully
          content:
            application/json:
              schema:
                type: object
                required:
                  - id
                  - name
                properties:
                  id:
                    type: string
                  name:
                    type: string
`;

const defaultRules = `rules:
  request.query.debug_enabled:
    enabled: true
    applies_to:
      - request
    type: query

    description: Detects requests that enable debug mode.

    match:
      paths:
        - "*"
      methods:
        - GET
      query_params:
        debug:
          - "true"

    finding:
      title: Debug query parameter enabled
      message: Request included debug=true in the query string.
`;

const defaultHeaders = {
  Accept: "application/json",
  "Content-Type": "application/json",
};

let editor;
let activeModelName = "openapi";

const models = {};
const viewStates = {};

require(["vs/editor/editor.main"], function () {
  initializeEditor();
});

function initializeEditor() {
  const editorElement = document.getElementById("playground-editor");

  if (!editorElement) {
    console.error("Could not find #playground-editor.");
    return;
  }

  models.openapi = monaco.editor.createModel(
    defaultOpenAPI,
    "yaml",
    monaco.Uri.parse("file:///openapi.yaml"),
  );

  models.rules = monaco.editor.createModel(
    defaultRules,
    "yaml",
    monaco.Uri.parse("file:///rules.yaml"),
  );

  editor = monaco.editor.create(editorElement, {
    model: models.openapi,
    theme: getEditorTheme(),
    automaticLayout: true,
    minimap: {
      enabled: false,
    },
    fontSize: 14,
    tabSize: 2,
    insertSpaces: true,
    scrollBeyondLastLine: false,
    wordWrap: "off",
  });

  window
    .matchMedia("(prefers-color-scheme: dark)")
    .addEventListener("change", function () {
      monaco.editor.setTheme(getEditorTheme());
    });
}

function getEditorTheme() {
  return window.matchMedia("(prefers-color-scheme: dark)").matches
    ? "vs-dark"
    : "vs";
}

function switchModel(modelName) {
  if (!editor || !models[modelName] || modelName === activeModelName) {
    return;
  }

  viewStates[activeModelName] = editor.saveViewState();

  activeModelName = modelName;
  editor.setModel(models[modelName]);

  if (viewStates[modelName]) {
    editor.restoreViewState(viewStates[modelName]);
  }

  document.querySelectorAll(".editor-tab").forEach(function (tab) {
    const isActive = tab.dataset.model === modelName;

    tab.classList.toggle("active", isActive);
    tab.setAttribute("aria-selected", String(isActive));
  });

  editor.focus();
}

function normalizePath(value) {
  const trimmed = value.trim();

  if (trimmed === "") {
    throw new Error("A request path is required.");
  }

  if (/^https?:\/\//i.test(trimmed)) {
    throw new Error(
      "Enter only a path, such as /users/123, not a complete URL.",
    );
  }

  return trimmed.startsWith("/") ? trimmed : `/${trimmed}`;
}

function buildPlaygroundPayload() {
  return {
    openapi: models.openapi.getValue(),
    rules: models.rules.getValue(),
    request: {
      method: document.getElementById("request-method").value,
      path: normalizePath(document.getElementById("request-path").value),
      headers: document.getElementById("request-headers").value,
      body: document.getElementById("request-body").value,
    },
  };
}

async function runAudit() {
  const runButton = document.getElementById("run-audit");

  if (!editor || !models.openapi || !models.rules) {
    setStatus("The editor is still loading.", "error");
    return;
  }

  let payload;

  try {
    payload = buildPlaygroundPayload();
  } catch (error) {
    setStatus(
      error instanceof Error ? error.message : "Invalid playground request.",
      "error",
    );

    return;
  }

  runButton.disabled = true;

  setStatus("Sending the simulated request...", "");

  renderAuditLoading();

  try {
    const response = await fetch("/playground", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      body: JSON.stringify(payload),
    });

    const result = await readResponse(response);

    if (!response.ok) {
      renderAuditError(result, response.status);

      throw new Error(getResponseErrorMessage(result, response.status));
    }

    renderAuditResult(result);
    setStatus("", "");
  } catch (error) {
    setStatus(
      error instanceof Error ? error.message : "The audit request failed.",
      "error",
    );
  } finally {
    runButton.disabled = false;
  }
}

async function readResponse(response) {
  const responseText = await response.text();

  if (responseText.trim() === "") {
    return {};
  }

  try {
    return JSON.parse(responseText);
  } catch {
    return {
      error: "invalid_server_response",
      message: responseText,
    };
  }
}

function getResponseErrorMessage(result, status) {
  if (
    result &&
    typeof result === "object" &&
    typeof result.message === "string"
  ) {
    return result.message;
  }

  if (
    result &&
    typeof result === "object" &&
    typeof result.error === "string"
  ) {
    return result.error;
  }

  return `The audit returned HTTP ${status}.`;
}

function setStatus(message, type) {
  const statusElement = document.getElementById("audit-status");

  statusElement.textContent = message;
  statusElement.classList.remove("error", "success");

  if (type) {
    statusElement.classList.add(type);
  }
}

function resetPlayground() {
  if (!models.openapi || !models.rules) {
    return;
  }

  models.openapi.setValue(defaultOpenAPI);
  models.rules.setValue(defaultRules);

  document.getElementById("request-method").value = "GET";

  document.getElementById("request-path").value = "/users/123?debug=true";

  document.getElementById("request-headers").value = JSON.stringify(
    defaultHeaders,
    null,
    2,
  );

  document.getElementById("request-body").value = "";

  resetAuditResult();
  switchModel("openapi");
}

document.querySelectorAll(".editor-tab").forEach(function (tab) {
  tab.addEventListener("click", function () {
    switchModel(tab.dataset.model);
  });
});

document.getElementById("run-audit").addEventListener("click", runAudit);

document
  .getElementById("reset-playground")
  .addEventListener("click", resetPlayground);
