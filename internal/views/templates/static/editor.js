const monacoBase = "https://cdn.jsdelivr.net/npm/monaco-editor@0.55.1/min/vs";

require.config({
  paths: {
    vs: monacoBase,
  },
});

const defaultOpenAPI = `openapi: 3.0.3

info:
  title: API Observer Investigation API
  version: 1.0.0
  description: >
    An API for reviewing findings and supporting evidence generated
    during API traffic analysis.

paths:
  /investigations/{id}:
    get:
      summary: Get an investigation
      description: >
        Returns the details and supporting evidence associated with
        an API traffic investigation.
      operationId: getInvestigation

      parameters:
        - name: id
          in: path
          required: true
          description: Unique investigation identifier.
          schema:
            type: string

        - name: include_evidence
          in: query
          required: false
          description: Include supporting evidence in the response.
          schema:
            type: boolean
            default: true

      responses:
        "200":
          description: Investigation returned successfully
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Investigation"

components:
  schemas:
    Investigation:
      type: object
      additionalProperties: false

      required:
        - id
        - title
        - status
        - confidence
        - severity
        - summary
        - evidence
        - recommendation

      properties:
        id:
          type: string

        title:
          type: string

        status:
          type: string
          enum:
            - open
            - requires_review
            - resolved

        confidence:
          type: number
          format: double
          minimum: 0
          maximum: 1

        severity:
          type: string
          enum:
            - informational
            - low
            - medium
            - high
            - critical

        summary:
          type: string

        evidence:
          type: array
          items:
            $ref: "#/components/schemas/Evidence"

        recommendation:
          type: string

    Evidence:
      type: object
      additionalProperties: false

      required:
        - type
        - description
        - url

      properties:
        type:
          type: string
          enum:
            - external_reference
            - request_capture
            - response_capture

        description:
          type: string

        url:
          type: string
          format: uri
`;

const defaultRules = `rules:
  request.query.include_evidence:
    enabled: true
    applies_to:
      - request
    type: query

    description: Detects requests that explicitly request supporting evidence.

    match:
      paths:
        - "/investigations/*"
      methods:
        - GET
      query_params:
        include_evidence:
          - "true"

    finding:
      title: Supporting evidence requested
      message: Request explicitly included supporting investigation evidence.
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
      "Enter only a path, such as /investigations/INC-2026-042, not a complete URL.",
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

  document.getElementById("request-path").value =
    "/investigations/INC-2026-042?include_evidence=true";

  document.getElementById("request-headers").value = JSON.stringify(
    defaultHeaders,
    null,
    2,
  );

  document.getElementById("request-body").value = "";

  resetAuditResult();
  setStatus("", "");
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
