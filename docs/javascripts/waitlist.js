firebase.initializeApp({
  apiKey: "AIzaSyBycaf1xkCsVAJauCCDJldBDsPBheocfTI",
  authDomain: "wdz-landingpage.firebaseapp.com",
  projectId: "wdz-landingpage",
  storageBucket: "wdz-landingpage.firebasestorage.app",
  messagingSenderId: "477988348745",
  appId: "1:477988348745:web:b15f8cc5ecf4d51e6f7098",
  measurementId: "G-BJR1BLBHC6",
});

const db = firebase.firestore();

document.querySelectorAll(".al-form").forEach((form) => {
  form.addEventListener("submit", async (e) => {
    e.preventDefault();

    const input = form.querySelector(".al-input");
    const btn = form.querySelector(".al-btn");
    const email = input.value.trim();

    if (!email) return;

    btn.disabled = true;
    btn.textContent = "Submitting...";

    try {
      await db.collection("agentledger-waitlist").add({
        email: email,
        timestamp: firebase.firestore.FieldValue.serverTimestamp(),
        source: window.location.href,
      });

      input.value = "";
      btn.textContent = "You're on the list!";
      btn.classList.add("al-btn--success");

      setTimeout(() => {
        btn.textContent = "Join the waitlist";
        btn.classList.remove("al-btn--success");
        btn.disabled = false;
      }, 3000);
    } catch (err) {
      console.error("Waitlist error:", err);
      btn.textContent = "Something went wrong";
      btn.classList.add("al-btn--error");

      setTimeout(() => {
        btn.textContent = "Join the waitlist";
        btn.classList.remove("al-btn--error");
        btn.disabled = false;
      }, 3000);
    }
  });
});
