const feed = document.getElementById("order-feed");
const statusDot = document.getElementById("status");

// Connect to WebSocket using current protocol
const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
const socket = new WebSocket(`${protocol}//${window.location.host}/ws`);

socket.onopen = () => statusDot.classList.add("online");
socket.onclose = () => statusDot.classList.remove("online");

socket.onmessage = function (event) {
  const newOrder = document.createElement("div");
  newOrder.className = "order-card";
  // Display raw server-formatted log
  newOrder.innerHTML = `<code>${event.data}</code>`;
  feed.prepend(newOrder);

  // Maintain a max of 10 entries for performance
  if (feed.children.length > 10) feed.removeChild(feed.lastChild);
};

document.getElementById("order-form").addEventListener("submit", async (e) => {
  e.preventDefault();

  const payload = {
    instrument: document.getElementById("instrument").value,
    quantity: parseInt(document.getElementById("quantity").value),
    price: parseFloat(document.getElementById("price").value),
  };

  try {
    const response = await fetch("/orders", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
    });

    if (response.ok) {
      e.target.reset(); // Clear form only on success
    } else {
      const err = await response.json();
      alert(`SYSTEM_REJECTED: ${err.error}`);
    }
  } catch (error) {
    alert("CRITICAL_CONNECTION_FAILURE");
  }
});
