const mainForm = document.getElementById("form");

const fileInput = document.getElementById("file");
const fileField = fileInput.closest(".field");

const contragentSelect = document.getElementById("counterparty");
const contragentField = contragentSelect.closest(".field");

const daytimeSelect = document.getElementById("time");
const daytimeField = daytimeSelect.closest(".field");

const worksheetList = document.getElementById("sheet");
const worksheetField = worksheetList.closest(".field");

const createInvoiceButton = document.getElementById("createInvoice");
const clearFormButton = document.getElementById("clearForm");
const copyTableButton = document.getElementById("copyTable");
const invoiceTitle = document.getElementById("invoiceTitle");
const invoiceCount = document.getElementById("invoiceCount");
const createInvoiceLoader = document.getElementById("createInvoiceLoader");
const loadingText = document.querySelector(
  "#createInvoiceLoader .loading-text",
);

const tbody = document.querySelector("#invoiceTable tbody");

let uploadRequestId = 0;
let isUploadingFile = false;
let isCreatingInvoice = false;
let isDeletingFile = false;
let lastCleanupInvoiceID = "";
const SELECT_ALL_WORKSHEETS_VALUE = "__all_worksheets__";
const SELECT_ALL_CONTRAGENTS_VALUE = "__all_contragents__";

function showError(fieldElement, message) {
  const warning = fieldElement.querySelector(".field-warning");

  if (!warning) return;

  warning.textContent = message;
  warning.classList.remove("success");
  warning.classList.add("is-visible");

  fieldElement.classList.remove("success");
  fieldElement.classList.add("error");
}

function showSuccess(fieldElement, message) {
  const warning = fieldElement.querySelector(".field-warning");

  if (!warning) return;

  warning.textContent = message;
  warning.classList.remove("is-visible");
  void warning.offsetWidth;

  warning.classList.add("is-visible", "success");

  fieldElement.classList.remove("error");
  fieldElement.classList.add("success");
}

function clearFieldState(fieldElement) {
  const warning = fieldElement.querySelector(".field-warning");

  if (!warning) return;

  warning.classList.remove("is-visible", "success");
  warning.textContent = "";

  fieldElement.classList.remove("error", "success");
}

async function parseJSON(response) {
  try {
    return await response.json();
  } catch (error) {
    return null;
  }
}

function resetUploadedFileState() {
  sessionStorage.removeItem("invoiceID");
  sessionStorage.removeItem("applicationType");
}

async function deleteFileByInvoiceID(invoiceID) {
  if (!invoiceID || isDeletingFile) return;

  isDeletingFile = true;

  try {
    await fetch("/delete_file", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        invoice_id: invoiceID,
      }),
    });
  } catch (error) {
    console.error(error.message);
  } finally {
    isDeletingFile = false;
  }
}

function cleanupUploadedFileOnPageLeave() {
  const invoiceID = sessionStorage.getItem("invoiceID");
  if (!invoiceID) return;

  if (invoiceID === lastCleanupInvoiceID) return;
  lastCleanupInvoiceID = invoiceID;

  const payload = JSON.stringify({
    invoice_id: invoiceID,
  });

  let requestSent = false;

  if (navigator.sendBeacon) {
    const body = new Blob([payload], { type: "application/json" });
    requestSent = navigator.sendBeacon("/delete_file", body);
  }

  if (!requestSent) {
    fetch("/delete_file", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: payload,
      keepalive: true,
    }).catch((error) => {
      console.error(error.message);
    });
  }

  resetUploadedFileState();
}

function setInvoiceTitle(title) {
  if (!invoiceTitle) return;
  invoiceTitle.textContent = typeof title === "string" ? title : "";
}

function setInvoiceCount(count) {
  if (!invoiceCount) return;
  invoiceCount.textContent = `Позиций: ${count}`;
}

function setLoadingState(isLoading, text = "") {
  if (createInvoiceLoader) {
    createInvoiceLoader.classList.toggle("is-visible", isLoading);
    createInvoiceLoader.setAttribute("aria-hidden", String(!isLoading));
  }
  if (loadingText && isLoading) {
    loadingText.textContent = text;
  }
}

function setWorksheetsDisabled(disabled) {
  worksheetList.classList.toggle("is-disabled", disabled);
  for (const input of worksheetList.querySelectorAll(
    'input[type="checkbox"]',
  )) {
    input.disabled = disabled;
  }
}

function renderWorksheets(worksheets = []) {
  worksheetList.innerHTML = "";

  if (!Array.isArray(worksheets) || worksheets.length === 0) {
    const placeholder = document.createElement("p");
    placeholder.className = "checkbox-list-placeholder";
    placeholder.textContent = "Выберите листы";
    worksheetList.appendChild(placeholder);
    setWorksheetsDisabled(true);
    return;
  }

  const columns = document.createElement("div");
  columns.className = "worksheet-columns";

  const leftColumn = document.createElement("div");
  leftColumn.className = "worksheet-column worksheet-column-left";

  const rightColumn = document.createElement("div");
  rightColumn.className = "worksheet-column worksheet-column-right";

  const selectAllOption = document.createElement("label");
  selectAllOption.className = "checkbox-option checkbox-option-select-all";

  const selectAllCheckbox = document.createElement("input");
  selectAllCheckbox.type = "checkbox";
  selectAllCheckbox.name = "sheet";
  selectAllCheckbox.value = SELECT_ALL_WORKSHEETS_VALUE;

  const selectAllText = document.createElement("span");
  selectAllText.textContent = "Выбрать все";

  selectAllOption.appendChild(selectAllCheckbox);
  selectAllOption.appendChild(selectAllText);
  leftColumn.appendChild(selectAllOption);

  for (const item of worksheets) {
    const option = document.createElement("label");
    option.className = "checkbox-option checkbox-option-worksheet";

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.name = "sheet";
    checkbox.value = item;

    const text = document.createElement("span");
    text.textContent = item;

    option.appendChild(checkbox);
    option.appendChild(text);
    rightColumn.appendChild(option);
  }

  columns.appendChild(leftColumn);
  columns.appendChild(rightColumn);
  worksheetList.appendChild(columns);
}

function getSelectedWorksheets() {
  const values = [];

  for (const input of worksheetList.querySelectorAll(
    'input[type="checkbox"]:checked',
  )) {
    if (input.value === SELECT_ALL_WORKSHEETS_VALUE) {
      continue;
    }

    values.push(input.value);
  }

  return values;
}

function syncSelectAllWorksheetsCheckbox() {
  const selectAllCheckbox = worksheetList.querySelector(
    `input[type="checkbox"][value="${SELECT_ALL_WORKSHEETS_VALUE}"]`,
  );
  if (!selectAllCheckbox) return;

  const worksheetCheckboxes = worksheetList.querySelectorAll(
    `input[type="checkbox"][name="sheet"]:not([value="${SELECT_ALL_WORKSHEETS_VALUE}"])`,
  );
  if (worksheetCheckboxes.length === 0) {
    selectAllCheckbox.checked = false;
    return;
  }

  selectAllCheckbox.checked = Array.from(worksheetCheckboxes).every(
    (checkbox) => checkbox.checked,
  );
}

function getInvoiceRows() {
  const rows = [];

  for (const row of tbody.querySelectorAll("tr")) {
    const cells = row.querySelectorAll("td");
    if (cells.length < 3) continue;

    const name = cells[0].textContent.trim();
    const article = cells[1].textContent.trim();
    const amount = cells[2].textContent.trim();

    if (!name && !article && !amount) continue;

    rows.push([name, article, amount]);
  }

  return rows;
}

function updateCopyButtonState() {
  if (!copyTableButton) return;
  copyTableButton.disabled = getInvoiceRows().length === 0;
}

async function copyTableToClipboard() {
  const rows = getInvoiceRows();
  if (rows.length === 0) {
    updateCopyButtonState();
    return;
  }

  const tableText = rows
    .map(([, article, amount]) => `${article}\t${amount}`)
    .join("\n");

  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(tableText);
    } else {
      const textarea = document.createElement("textarea");
      textarea.value = tableText;
      textarea.setAttribute("readonly", "");
      textarea.style.position = "fixed";
      textarea.style.top = "-9999px";
      document.body.appendChild(textarea);
      textarea.select();

      const copied = document.execCommand("copy");
      document.body.removeChild(textarea);
      if (!copied) {
        throw new Error("Copy command failed");
      }
    }

    showSuccess(fileField, "Таблица скопирована");
  } catch (error) {
    showError(fileField, "Ошибка: не удалось скопировать таблицу");
    console.error(error.message);
  }
}

function clearForm(options = {}) {
  const { resetFile = false, deleteRemote = false } = options;
  const invoiceID = sessionStorage.getItem("invoiceID");

  contragentSelect.disabled = true;
  while (contragentSelect.options.length > 1) {
    contragentSelect.remove(1);
  }

  daytimeSelect.disabled = true;
  while (daytimeSelect.options.length > 1) {
    daytimeSelect.remove(1);
  }

  renderWorksheets();

  createInvoiceButton.disabled = true;

  resetUploadedFileState();

  if (deleteRemote && invoiceID) {
    deleteFileByInvoiceID(invoiceID);
  }

  if (resetFile) {
    fileInput.value = "";
  }

  clearFieldState(fileField);
  clearFieldState(contragentField);
  clearFieldState(daytimeField);
  clearFieldState(worksheetField);

  tbody.innerHTML = "";
  setInvoiceTitle("");
  setInvoiceCount(0);
  updateCopyButtonState();
}

fileInput.addEventListener("change", async function (event) {
  if (isUploadingFile) return;

  clearForm({ deleteRemote: true });
  const file = event.target.files[0];
  if (!file) return;
  const currentRequestId = ++uploadRequestId;

  const formData = new FormData();
  formData.append("file", file);
  formData.append("manufacture_type", "kond");

  isUploadingFile = true;
  setLoadingState(true, "Загружаем файл...");

  try {
    const response = await fetch("/kond/upload_file", {
      method: "POST",
      body: formData,
    });
    const result = await parseJSON(response);
    if (currentRequestId !== uploadRequestId) return;

    if (!response.ok) {
      showError(
        fileField,
        `Ошибка: ${result?.error ?? "Не удалось загрузить файл"}`,
      );
      return;
    }

    if (!result?.id || !result?.application_type) {
      showError(fileField, "Ошибка: сервер вернул некорректные данные");
      return;
    }

    sessionStorage.setItem("invoiceID", result.id);
    sessionStorage.setItem("applicationType", result.application_type);

    contragentSelect.disabled = false;
    createInvoiceButton.disabled = false;

    const contrAgents = Array.isArray(result.contr_agents)
      ? result.contr_agents
      : [];
    const daytimes = Array.isArray(result.daytimes) ? result.daytimes : [];
    const worksheets = Array.isArray(result.worksheets)
      ? result.worksheets
      : [];

    if (result.application_type === "store") {
      const allOption = document.createElement("option");
      allOption.value = SELECT_ALL_CONTRAGENTS_VALUE;
      allOption.textContent = "Все контрагенты";
      contragentSelect.appendChild(allOption);
    }

    for (const item of contrAgents) {
      const option = document.createElement("option");
      option.value = item;
      option.textContent = item;

      contragentSelect.appendChild(option);
    }

    renderWorksheets(worksheets);
    setWorksheetsDisabled(false);

    if (daytimes.length > 0) {
      daytimeSelect.disabled = false;

      for (const item of daytimes) {
        const option = document.createElement("option");
        option.value = item;
        option.textContent = item;

        daytimeSelect.appendChild(option);
      }
    }

    showSuccess(fileField, `Файл ${file.name} загружен`);
  } catch (error) {
    if (currentRequestId === uploadRequestId) {
      showError(
        fileField,
        "Ошибка: не удалось загрузить файл. Проверьте соединение",
      );
    }
    console.error(error.message);
  } finally {
    isUploadingFile = false;
    setLoadingState(false);
  }
});

mainForm.addEventListener("submit", async function (event) {
  event.preventDefault();

  let flag = false;

  const contragent = contragentSelect.value;
  const daytime = daytimeSelect.value;
  const worksheets = getSelectedWorksheets();

  if (contragent === "") {
    showError(contragentField, "Выберите контрагента");
    flag = true;
  }

  if (!daytimeSelect.disabled && daytime === "") {
    showError(daytimeField, "Выберите время");
    flag = true;
  }

  if (worksheets.length === 0) {
    showError(worksheetField, "Выберите хотя бы один лист");
    flag = true;
  }

  if (flag) return;
  if (isCreatingInvoice) return;

  const invoiceID = sessionStorage.getItem("invoiceID");
  const applicationType = sessionStorage.getItem("applicationType");

  if (!invoiceID || !applicationType) {
    showError(fileField, "Сначала загрузите файл");
    return;
  }

  const request = {
    invoice_id: invoiceID,
    contr_agent: contragent,
    daytime: daytime,
    worksheets: worksheets,
    application_type: applicationType,
  };
  let createInvoiceEndpoint = "/kond/create_invoice";

  if (contragent === SELECT_ALL_CONTRAGENTS_VALUE) {
    createInvoiceEndpoint = "/kond/create_invoice_all_contragents";
    delete request.contr_agent;
  }

  isCreatingInvoice = true;
  setLoadingState(true, "Создаем накладную...");

  try {
    const response = await fetch(createInvoiceEndpoint, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      body: JSON.stringify(request),
    });

    const result = await parseJSON(response);

    if (!response.ok) {
      showError(
        fileField,
        `Ошибка: ${result?.error ?? "Не удалось создать накладную"}`,
      );
      return;
    }

    if (!result || !Array.isArray(result.invoice)) {
      showError(fileField, "Ошибка: сервер вернул некорректные данные");
      return;
    }

    console.log(result);

    tbody.innerHTML = "";

    for (const value of result.invoice) {
      if (!Array.isArray(value) || value.length < 3) {
        continue;
      }

      const row = document.createElement("tr");

      const nameCell = document.createElement("td");
      nameCell.textContent = value[0];

      const articleCell = document.createElement("td");
      articleCell.textContent = value[1];

      const amountCell = document.createElement("td");
      amountCell.textContent = value[2];

      row.appendChild(nameCell);
      row.appendChild(articleCell);
      row.appendChild(amountCell);

      tbody.appendChild(row);
    }

    setInvoiceTitle(result.title ?? "");
    setInvoiceCount(result.invoice.length);
    updateCopyButtonState();
    showSuccess(fileField, "Накладная успешно создана");
  } catch (error) {
    showError(
      fileField,
      "Ошибка: не удалось создать накладную. Проверьте соединение",
    );
    console.error(error.message);
  } finally {
    isCreatingInvoice = false;
    setLoadingState(false);
  }
});

clearFormButton.addEventListener("click", () => {
  clearForm({ resetFile: true, deleteRemote: true });
});

if (copyTableButton) {
  copyTableButton.addEventListener("click", copyTableToClipboard);
}

contragentSelect.addEventListener("change", () => {
  clearFieldState(contragentField);
});

daytimeSelect.addEventListener("change", () => {
  clearFieldState(daytimeField);
});

worksheetList.addEventListener("change", (event) => {
  const target = event.target;
  if (!target.matches('input[type="checkbox"]')) return;

  if (target.value === SELECT_ALL_WORKSHEETS_VALUE) {
    for (const checkbox of worksheetList.querySelectorAll(
      `input[type="checkbox"][name="sheet"]:not([value="${SELECT_ALL_WORKSHEETS_VALUE}"])`,
    )) {
      checkbox.checked = target.checked;
    }
  } else {
    syncSelectAllWorksheetsCheckbox();
  }

  clearFieldState(worksheetField);
});

updateCopyButtonState();
setInvoiceCount(0);

window.addEventListener("beforeunload", cleanupUploadedFileOnPageLeave);
window.addEventListener("pagehide", cleanupUploadedFileOnPageLeave);
