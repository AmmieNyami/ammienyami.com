(() => {
  const plainTextContent = document.getElementById("plainTextContent");

  const searchParams = new URLSearchParams(window.location.search);
  const filePath = searchParams.get("file");
  if (!filePath) {
    plainTextContent.textContent =
      "Please specify a file path in the `file` URL parameter.";
    return;
  }

  fetch(filePath)
    .then((res) =>
      res.text().then((text) => {
        plainTextContent.textContent = text;
      }),
    )
    .catch(() => {
      plainTextContent.textContent =
        "Failed to fetch the file in the `file` URL parameter.";
    });
})();
