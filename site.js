// Withdo site: download links to the latest release, the showcase tabs, the live agent card, entrances.
(function () {
  var ua = navigator.userAgent;
  // iPhones and iPads say "like Mac OS X"; there is no iOS app, so they get no default.
  var os = /Android/i.test(ua) ? "android" : /iPhone|iPad|iPod/i.test(ua) || (/Mac/i.test(ua) && navigator.maxTouchPoints > 1) ? ""
    : /Windows/i.test(ua) ? "win" : /Mac/i.test(ua) ? "mac" : "";
  var names = { mac: "macOS", win: "Windows", android: "Android" };
  // Each platform: the file of the latest release whose name starts and ends like this. The APK is
  // withdo-<v>.apk from 0.12.0 and adhd-todo-<v>.apk before.
  var files = { mac: [["Withdo-", "-macos-universal.dmg"]], win: [["Withdo-Setup-", "-windows-x64.exe"]],
                android: [["withdo-", ".apk"], ["adhd-todo-", ".apk"]] };

  if (os) {
    document.querySelectorAll('.btn[data-dl="auto"], .cells [data-dl="auto"]').forEach(function (a) {
      a.textContent = "Download for " + names[os];
    });
    var note = document.querySelector("[data-dlnote]");
    if (note) note.textContent = "Download Withdo for " + names[os] + ". It is free, and it updates itself.";
  }

  fetch("https://api.github.com/repos/QingYunA/withdo/releases/latest")
    .then(function (r) { return r.ok ? r.json() : null; })
    .then(function (rel) {
      if (!rel || !rel.assets) return;
      var url = {};
      Object.keys(files).forEach(function (k) {
        files[k].some(function (f) {
          var hit = rel.assets.filter(function (x) {
            return x.name.indexOf(f[0]) === 0 && x.name.slice(-f[1].length) === f[1];
          })[0];
          if (hit) url[k] = hit.browser_download_url;
          return !!hit;
        });
      });
      document.querySelectorAll("[data-dl]").forEach(function (a) {
        var k = a.dataset.dl === "auto" ? os : a.dataset.dl;
        if (url[k]) a.href = url[k];
      });
      var v = document.getElementById("ver");
      if (v) v.textContent = "Withdo " + rel.tag_name.replace(/^v/, "") + " · open source under the AGPL-3.0";
    })
    .catch(function () {});

  // Showcase tabs
  var text = {
    film: "Twenty-three seconds: four agents, too many questions at once, and the one board that sorts them out.",
    today: "Your tasks for today, and beside them the agent cards that need you: a reply to write, a pull request to review.",
    agents: "Every agent's work, by agent: the branch, the step it is on now, and which one has been blocked for three days.",
    phone: "The same tasks on Android, kept in step with your Mac or PC once you sign in."
  };
  var tabs = Array.prototype.slice.call(document.querySelectorAll("[data-tab]"));
  var film = document.getElementById("film");
  function show(name, focus) {
    tabs.forEach(function (t) {
      var on = t.dataset.tab === name;
      t.setAttribute("aria-selected", on);
      t.tabIndex = on ? 0 : -1;
      if (on && focus) t.focus();
    });
    document.querySelectorAll("[data-pane]").forEach(function (p) { p.hidden = p.dataset.pane !== name; });
    document.getElementById("pane-text").textContent = text[name];
    if (name === "film") { film.play().catch(function () {}); } else { film.pause(); }
  }
  tabs.forEach(function (t, i) {
    t.addEventListener("click", function () { show(t.dataset.tab); });
    t.addEventListener("keydown", function (e) {
      var d = e.key === "ArrowRight" ? 1 : e.key === "ArrowLeft" ? -1 : 0;
      if (d) { e.preventDefault(); show(tabs[(i + d + tabs.length) % tabs.length].dataset.tab, true); }
    });
  });
  var sound = document.getElementById("sound");
  sound.addEventListener("click", function () {
    film.muted = !film.muted;
    if (!film.muted) { film.currentTime = 0; film.play().catch(function () {}); }
    sound.textContent = film.muted ? "Sound on" : "Sound off";
    sound.setAttribute("aria-pressed", String(!film.muted));
  });

  // The live card: answer Codex
  var status = document.getElementById("codex-status");
  var q = document.getElementById("codex-q");
  var answers = document.getElementById("codex-answers");
  var card = document.getElementById("codex");
  function answer(choice) {
    status.textContent = "Doing";
    status.className = "chip-s doing notch";
    q.textContent = "You answered " + choice + ". Adding a " + choice + " cache now.";
    answers.hidden = true;
    var again = document.createElement("button");
    again.type = "button"; again.className = "again"; again.textContent = "Ask again";
    again.addEventListener("click", function () {
      status.textContent = "Blocked"; status.className = "chip-s blocked notch";
      q.textContent = "Which database should the cache use?";
      answers.hidden = false; again.remove(); answers.querySelector("button").focus();
    });
    card.appendChild(again);
    card.classList.remove("flash"); void card.offsetWidth; card.classList.add("flash");
    again.focus();
  }
  answers.querySelectorAll("button").forEach(function (b) {
    b.addEventListener("click", function () { answer(b.dataset.a); });
  });

  // Entrances, and the shutter on the showcase plate
  // The plate is observed through its wrapper: a fully clipped element never counts as visible.
  var plate = document.getElementById("plate");
  var wrap = plate.parentNode;
  var targets = Array.prototype.slice.call(document.querySelectorAll("[data-in]")).concat([wrap]);
  if (!("IntersectionObserver" in window)) {
    targets.forEach(function (el) { el.classList.add("on"); });
    plate.classList.remove("shut");
    return;
  }
  var io = new IntersectionObserver(function (entries) {
    entries.forEach(function (e) {
      if (!e.isIntersecting) return;
      e.target.classList.add("on");
      if (e.target === wrap) { plate.classList.remove("shut"); film.play().catch(function () {}); }
      io.unobserve(e.target);
    });
  }, { threshold: 0.15 });
  targets.forEach(function (el) { io.observe(el); });
})();
