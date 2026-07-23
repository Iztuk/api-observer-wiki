function renderAuditLoading() {
  const resultElement = getAuditResultElement();

  resultElement.replaceChildren();

  const loading = document.createElement("p");
  loading.className = "empty-result";
  loading.textContent = "Running audit...";

  resultElement.appendChild(loading);
}

function resetAuditResult() {
  const resultElement = getAuditResultElement();

  resultElement.replaceChildren();

  const empty = document.createElement("p");
  empty.className = "empty-result";
  empty.textContent = "Run the audit to view findings.";

  resultElement.appendChild(empty);
}

function renderAuditError(result, statusCode) {
  const resultElement = getAuditResultElement();

  resultElement.replaceChildren();

  const errorCard = document.createElement("div");
  errorCard.className = "audit-error-card";

  const title = document.createElement("strong");
  title.textContent = `Audit failed (${statusCode})`;

  const message = document.createElement("p");
  message.textContent = getAuditErrorMessage(result);

  errorCard.append(title, message);
  resultElement.appendChild(errorCard);

  resultElement.appendChild(buildRawJSONSection(result));
}

function renderAuditResult(result) {
  const resultElement = getAuditResultElement();

  resultElement.replaceChildren();

  const jobs = getAuditJobs(result);
  const findings = getFindings(result);

  resultElement.appendChild(buildAuditSummary(jobs, findings));

  resultElement.appendChild(buildFindingsSection(findings));

  if (jobs.length > 0) {
    resultElement.appendChild(buildJobsSection(jobs));
  }

  resultElement.appendChild(buildRawJSONSection(result));
}

function getAuditResultElement() {
  const resultElement = document.getElementById("audit-result");

  if (!resultElement) {
    throw new Error("Could not find #audit-result.");
  }

  return resultElement;
}

function getAuditJobs(result) {
  if (Array.isArray(result.audit_jobs)) {
    return result.audit_jobs;
  }

  if (Array.isArray(result.auditJob)) {
    return result.auditJob;
  }

  if (Array.isArray(result.jobs)) {
    return result.jobs;
  }

  if (Array.isArray(result.job)) {
    return result.job;
  }

  return [];
}

function getFindings(result) {
  return Array.isArray(result.findings) ? result.findings : [];
}

function buildAuditSummary(jobs, findings) {
  const summary = document.createElement("div");
  summary.className = "audit-summary";

  const content = document.createElement("div");

  const title = document.createElement("strong");
  title.textContent =
    findings.length > 0 ? "Audit completed with findings" : "Audit completed";

  const description = document.createElement("p");

  description.textContent =
    `${findings.length} ${pluralize("finding", findings.length)} · ` +
    `${jobs.length} ${pluralize("job", jobs.length)}`;

  content.append(title, description);

  const badge = document.createElement("span");

  badge.className =
    findings.length > 0 ? "summary-badge warning" : "summary-badge success";

  badge.textContent =
    findings.length > 0 ? `${findings.length} found` : "Passed";

  summary.append(content, badge);

  return summary;
}

function buildFindingsSection(findings) {
  const section = document.createElement("section");
  section.className = "result-section";

  const heading = document.createElement("h4");
  heading.textContent = "Findings";

  section.appendChild(heading);

  if (findings.length === 0) {
    const empty = document.createElement("div");
    empty.className = "no-findings";
    empty.textContent = "No findings were generated.";

    section.appendChild(empty);

    return section;
  }

  const list = document.createElement("div");
  list.className = "findings-list";

  for (const finding of findings) {
    list.appendChild(buildFindingCard(finding));
  }

  section.appendChild(list);

  return section;
}

function buildFindingCard(finding) {
  const card = document.createElement("article");
  card.className = "finding-card";

  const header = document.createElement("div");
  header.className = "finding-header";

  const title = document.createElement("h5");
  title.textContent = finding.title || "Untitled finding";

  const ruleID = document.createElement("code");
  ruleID.textContent = finding.rule_id || "unknown_rule";

  header.append(title, ruleID);

  const message = document.createElement("p");
  message.className = "finding-message";

  message.textContent = finding.message || "No finding message was provided.";

  const metadata = document.createElement("dl");
  metadata.className = "finding-metadata";

  appendDefinition(metadata, "Job", shortenID(finding.job_id));

  appendDefinition(metadata, "Finding ID", shortenID(finding.id));

  appendDefinition(metadata, "Created", formatTimestamp(finding.created_at));

  card.append(header, message, metadata);

  return card;
}

function buildJobsSection(jobs) {
  const section = document.createElement("section");
  section.className = "result-section";

  const heading = document.createElement("h4");
  heading.textContent = "Audit jobs";

  const list = document.createElement("div");
  list.className = "jobs-list";

  for (const job of jobs) {
    list.appendChild(buildJobCard(job));
  }

  section.append(heading, list);

  return section;
}

function buildJobCard(job) {
  const details = document.createElement("details");
  details.className = "job-card";

  const summary = document.createElement("summary");
  summary.className = "job-summary";

  const method = document.createElement("span");
  method.className = "method-badge";
  method.textContent = job.method || "UNKNOWN";

  const target = document.createElement("code");
  target.textContent = buildRequestTarget(job);

  summary.append(method, target);

  const content = document.createElement("div");
  content.className = "job-content";

  const metadata = document.createElement("dl");
  metadata.className = "job-metadata";

  appendDefinition(metadata, "Host", job.host || "—");

  appendDefinition(metadata, "Type", job.type || "—");

  appendDefinition(metadata, "Status", formatStatus(job.status));

  appendDefinition(metadata, "Request ID", shortenID(job.request_id));

  appendDefinition(metadata, "Timestamp", formatTimestamp(job.timestamp));

  content.appendChild(metadata);

  if (job.headers && Object.keys(job.headers).length > 0) {
    content.appendChild(buildHeadersSection(job.headers));
  }

  if (job.body !== undefined && job.body !== "") {
    content.appendChild(buildCodeSection("Body", formatBody(job.body)));
  }

  if (job.error) {
    const error = document.createElement("p");
    error.className = "job-error";
    error.textContent = job.error;

    content.appendChild(error);
  }

  details.append(summary, content);

  return details;
}

function buildHeadersSection(headers) {
  const section = document.createElement("div");
  section.className = "job-data-section";

  const heading = document.createElement("h6");
  heading.textContent = "Headers";

  const list = document.createElement("dl");
  list.className = "header-list";

  for (const [name, value] of Object.entries(headers)) {
    appendDefinition(
      list,
      name,
      Array.isArray(value) ? value.join(", ") : String(value),
    );
  }

  section.append(heading, list);

  return section;
}

function buildCodeSection(titleText, value) {
  const section = document.createElement("div");
  section.className = "job-data-section";

  const heading = document.createElement("h6");
  heading.textContent = titleText;

  const pre = document.createElement("pre");
  const code = document.createElement("code");

  code.textContent = value;

  pre.appendChild(code);
  section.append(heading, pre);

  return section;
}

function buildRawJSONSection(result) {
  const details = document.createElement("details");
  details.className = "raw-json";

  const summary = document.createElement("summary");
  summary.textContent = "Raw JSON";

  const pre = document.createElement("pre");
  const code = document.createElement("code");

  code.textContent = JSON.stringify(result, null, 2);

  pre.appendChild(code);
  details.append(summary, pre);

  return details;
}

function appendDefinition(list, label, value) {
  const term = document.createElement("dt");
  term.textContent = label;

  const description = document.createElement("dd");

  description.textContent = value ?? "—";

  list.append(term, description);
}

function buildRequestTarget(job) {
  const path = job.path || "/";
  const query = job.query ? `?${job.query}` : "";

  return `${path}${query}`;
}

function formatBody(body) {
  if (typeof body !== "string") {
    return JSON.stringify(body, null, 2);
  }

  try {
    return JSON.stringify(JSON.parse(body), null, 2);
  } catch {
    return body;
  }
}

function formatTimestamp(value) {
  if (!value) {
    return "—";
  }

  const date = new Date(value);

  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString();
}

function formatStatus(status) {
  if (status === undefined || status === null || status === 0) {
    return "—";
  }

  return String(status);
}

function shortenID(value) {
  if (!value) {
    return "—";
  }

  return value.length > 12 ? `${value.slice(0, 8)}…` : value;
}

function pluralize(word, amount) {
  return amount === 1 ? word : `${word}s`;
}

function getAuditErrorMessage(result) {
  if (result && typeof result.message === "string") {
    return result.message;
  }

  if (result && typeof result.error === "string") {
    return result.error;
  }

  return "The audit request failed.";
}
