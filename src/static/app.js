/* ─── theme ─── */
(() => {
	const stored = localStorage.getItem("lustre-theme");
	if (
		stored === "dark" ||
		(!stored && matchMedia("(prefers-color-scheme:dark)").matches)
	) {
		document.documentElement.classList.add("dark");
	}
})();

document.getElementById("btn-theme").addEventListener("click", () => {
	document.documentElement.classList.toggle("dark");
	localStorage.setItem(
		"lustre-theme",
		document.documentElement.classList.contains("dark") ? "dark" : "light",
	);
});

/* ─── rendering ─── */
function esc(s) {
	return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}

function renderLineContent(fullLine, changes, side) {
	if (!changes || changes.length === 0) {
		return `<span class="tok-normal">${esc(fullLine)}</span>`;
	}
	const changeCls = side === "add" ? "tok-change-add" : "tok-change-rem";
	const sorted = [...changes].sort((a, b) => a.start - b.start);
	let html = "",
		cursor = 0;
	for (const ch of sorted) {
		if (ch.start > cursor)
			html += `<span class="tok-normal">${esc(fullLine.slice(cursor, ch.start))}</span>`;
		html += `<span class="tok-${ch.highlight} ${changeCls}">${esc(ch.content)}</span>`;
		cursor = ch.end;
	}
	if (cursor < fullLine.length)
		html += `<span class="tok-normal">${esc(fullLine.slice(cursor))}</span>`;
	return html;
}

function renderFile(file) {
	const diff = file.diff;
	if (!diff || diff === null || !diff.aligned_lines) {
		// Added or removed file without structural diff — show raw
		return renderPlainFile(file);
	}

	const oldLines = (file.old_src || "").split("\n");
	const newLines = (file.new_src || "").split("\n");

	const lhsChanges = new Map(),
		rhsChanges = new Map();
	const lhsChanged = new Set(),
		rhsChanged = new Set();

	if (diff.chunks) {
		for (const chunk of diff.chunks) {
			for (const entry of chunk) {
				if (entry.lhs) {
					const ln = entry.lhs.line_number;
					if (entry.lhs.changes?.length) {
						lhsChanges.set(
							ln,
							(lhsChanges.get(ln) || []).concat(entry.lhs.changes),
						);
						lhsChanged.add(ln);
					}
				}
				if (entry.rhs) {
					const ln = entry.rhs.line_number;
					if (entry.rhs.changes?.length) {
						rhsChanges.set(
							ln,
							(rhsChanges.get(ln) || []).concat(entry.rhs.changes),
						);
						rhsChanged.add(ln);
					}
				}
				if (entry.rhs && !entry.lhs) rhsChanged.add(entry.rhs.line_number);
				if (entry.lhs && !entry.rhs) lhsChanged.add(entry.lhs.line_number);
			}
		}
	}

	const addedLines = new Set(),
		removedLines = new Set();
	for (const [o, n] of diff.aligned_lines) {
		if (o === null) addedLines.add(n);
		if (n === null) removedLines.add(o);
	}

	let rows = "";
	for (const [oldLn, newLn] of diff.aligned_lines) {
		const isAdded = oldLn === null;
		const isRemoved = newLn === null;
		const isLhsChanged = oldLn !== null && lhsChanged.has(oldLn);
		const isRhsChanged = newLn !== null && rhsChanged.has(newLn);

		if (isAdded) {
			const t = newLines[newLn] || "";
			const ch = rhsChanges.get(newLn);
			const code = ch
				? renderLineContent(t, ch, "add")
				: `<span class="tok-normal">${esc(t)}</span>`;
			rows +=
				`<tr><td class="gutter"></td><td class="code empty-cell"></td>` +
				`<td class="divider"></td>` +
				`<td class="gutter added">${newLn + 1}</td><td class="code added">${code}</td></tr>`;
			continue;
		}
		if (isRemoved) {
			const t = oldLines[oldLn] || "";
			const ch = lhsChanges.get(oldLn);
			const code = ch
				? renderLineContent(t, ch, "rem")
				: `<span class="tok-normal">${esc(t)}</span>`;
			rows +=
				`<tr><td class="gutter removed">${oldLn + 1}</td><td class="code removed">${code}</td>` +
				`<td class="divider"></td>` +
				`<td class="gutter"></td><td class="code empty-cell"></td></tr>`;
			continue;
		}

		const lt = oldLines[oldLn] || "",
			rt = newLines[newLn] || "";
		const lc = lhsChanges.get(oldLn),
			rc = rhsChanges.get(newLn);
		const leftCode = lc
			? renderLineContent(lt, lc, "rem")
			: `<span class="tok-normal">${esc(lt)}</span>`;
		const rightCode = rc
			? renderLineContent(rt, rc, "add")
			: `<span class="tok-normal">${esc(rt)}</span>`;
		const lCls = isLhsChanged ? " removed" : "";
		const rCls = isRhsChanged ? " added" : "";

		rows +=
			`<tr>` +
			`<td class="gutter${lCls}">${oldLn + 1}</td><td class="code${lCls}">${leftCode}</td>` +
			`<td class="divider"></td>` +
			`<td class="gutter${rCls}">${newLn + 1}</td><td class="code${rCls}">${rightCode}</td>` +
			`</tr>`;
	}

	return { html: rows, added: addedLines.size, removed: removedLines.size };
}

function renderPlainFile(file) {
	const lines = (
		file.status === "added" ? file.new_src : file.old_src || ""
	).split("\n");
	let rows = "";
	for (let i = 0; i < lines.length; i++) {
		const code = `<span class="tok-normal">${esc(lines[i])}</span>`;
		if (file.status === "added") {
			rows +=
				`<tr><td class="gutter"></td><td class="code empty-cell"></td>` +
				`<td class="divider"></td>` +
				`<td class="gutter added">${i + 1}</td><td class="code added">${code}</td></tr>`;
		} else {
			rows +=
				`<tr><td class="gutter removed">${i + 1}</td><td class="code removed">${code}</td>` +
				`<td class="divider"></td>` +
				`<td class="gutter"></td><td class="code empty-cell"></td></tr>`;
		}
	}
	return {
		html: rows,
		added: file.status === "added" ? lines.length : 0,
		removed: file.status === "removed" ? lines.length : 0,
	};
}

function renderAll(data) {
	const diffsEl = document.getElementById("diffs");
	const filesEl = document.getElementById("sb-files");
	const loading = document.getElementById("loading");

	if (!data.files || data.files.length === 0) {
		loading.style.display = "none";
		diffsEl.innerHTML =
			'<div class="empty-state"><div class="big">✓</div>No changes between branches.</div>';
		filesEl.innerHTML = "";
		document.getElementById("sb-count").textContent = "0";
		document.getElementById("sb-total").innerHTML = "";
		return;
	}

	loading.style.display = "none";
	let totalAdded = 0,
		totalRemoved = 0;
	let diffsHTML = "";
	let filesHTML = "";

	for (let i = 0; i < data.files.length; i++) {
		const file = data.files[i];
		const result = renderFile(file);
		totalAdded += result.added;
		totalRemoved += result.removed;

		const lang = file.diff?.language || file.language || "";

		// sidebar entry
		const statusIcon =
			file.status === "added" ? "A" : file.status === "removed" ? "D" : "M";
		const statusCls =
			file.status === "added" ? "a" : file.status === "removed" ? "r" : "c";
		filesHTML +=
			`<div class="sb-file" data-idx="${i}">` +
			`<span class="icon ${statusCls}">${statusIcon}</span>` +
			`<span class="name">${esc(file.path)}</span>` +
			`<span class="stats"><span class="a">+${result.added}</span> <span class="r">-${result.removed}</span></span>` +
			`</div>`;

		// diff card
		diffsHTML +=
			`<div class="diff-card" id="file-${i}">` +
			`<div class="diff-header">` +
			`<span class="filepath">${esc(file.path)}</span>` +
			(lang ? `<span class="lang-badge">${esc(lang)}</span>` : "") +
			`<span class="stat"><span class="add">+${result.added}</span>&ensp;<span class="rem">−${result.removed}</span></span>` +
			`</div>` +
			`<div class="diff-container"><table class="diff"><colgroup>` +
			`<col class="gutter"><col class="code"><col style="width:1px"><col class="gutter"><col class="code">` +
			`</colgroup><tbody>${result.html}</tbody></table></div>` +
			`</div>`;
	}

	diffsEl.innerHTML = diffsHTML;
	filesEl.innerHTML = filesHTML;
	document.getElementById("sb-count").textContent = data.files.length;
	document.getElementById("sb-total").innerHTML =
		`<b>+${totalAdded}</b> / <b>-${totalRemoved}</b>`;

	// sidebar click → scroll
	filesEl.querySelectorAll(".sb-file").forEach((el) => {
		el.addEventListener("click", () => {
			const idx = el.dataset.idx;
			document
				.getElementById(`file-${idx}`)
				.scrollIntoView({ behavior: "smooth", block: "start" });
		});
	});

	// intersection observer for active sidebar highlight
	setupScrollSpy(data.files.length);
}

function setupScrollSpy(count) {
	const mainEl = document.getElementById("main");
	const observer = new IntersectionObserver(
		(entries) => {
			for (const entry of entries) {
				if (entry.isIntersecting) {
					const idx = entry.target.id.replace("file-", "");
					document.querySelectorAll(".sb-file").forEach((el) => {
						el.classList.remove("active");
					});
					const active = document.querySelector(`.sb-file[data-idx="${idx}"]`);
					if (active) active.classList.add("active");
				}
			}
		},
		{ root: mainEl, rootMargin: "-10% 0px -80% 0px", threshold: 0 },
	);

	for (let i = 0; i < count; i++) {
		const el = document.getElementById(`file-${i}`);
		if (el) observer.observe(el);
	}
}

/* ─── data fetching ─── */
async function fetchDiffs() {
	try {
		const res = await fetch("/api/diffs");
		const data = await res.json();
		renderAll(data);
	} catch (e) {
		console.error("fetch failed:", e);
	}
}

/* ─── SSE ─── */
let currentSource = null;
let reconnectTimer = null;

function connectSSE() {
	const indicator = document.getElementById("sb-indicator");

	// Clean up any previous connection.
	if (currentSource) {
		currentSource.close();
		currentSource = null;
	}
	if (reconnectTimer !== null) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}

	const evtSource = new EventSource("/events");
	currentSource = evtSource;

	evtSource.addEventListener("connected", () => {
		indicator.classList.add("live");
	});

	evtSource.addEventListener("reload", () => {
		fetchDiffs();
	});

	evtSource.onerror = () => {
		indicator.classList.remove("live");
		evtSource.close();
		currentSource = null;
		// Reconnect after a delay; the guard above prevents stacking.
		reconnectTimer = setTimeout(() => {
			reconnectTimer = null;
			connectSSE();
		}, 2000);
	};
}

/* ─── init ─── */
fetchDiffs();
connectSSE();
