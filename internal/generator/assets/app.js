(function() {
  'use strict';

  const data = window.COVERAGE_DATA;
  const config = window.COVERAGE_CONFIG || { syntaxEnabled: true };

  // State
  let currentFileId = null;
  let searchQuery = '';
  let contentSearchQuery = '';
  let matches = [];
  let currentMatchIndex = -1;
  let expandedDirs = new Set();
  let syntaxHighlightEnabled = config.syntaxEnabled;
  let sortMode = 'name'; // 'name' or 'coverage'
  let anchorLine = null;        // First line clicked (anchor for shift-select)
  let selectedRange = null;     // { start: N, end: M } or null

  // DOM elements
  const fileTree = document.getElementById('file-tree');
  const viewport = document.getElementById('viewport');
  const filePath = document.getElementById('file-path');
  const summary = document.getElementById('summary');
  const searchInput = document.getElementById('search-input');
  const contentSearch = document.getElementById('content-search');
  const matchInfo = document.getElementById('match-info');
  const prevMatch = document.getElementById('prev-match');
  const nextMatch = document.getElementById('next-match');
  const themeToggle = document.getElementById('theme-toggle');
  const syntaxToggle = document.getElementById('syntax-toggle');
  const helpModal = document.getElementById('help-modal');
  const closeHelp = document.getElementById('close-help');
  const helpToggle = document.getElementById('help-toggle');

  // Coverage cache: fileId -> percentage
  let coverageCache = new Map();

  function initCoverageCache() {
    data.files.forEach((file, idx) => {
      coverageCache.set(idx, calculateFileCoverage(idx));
    });
  }

  function calculateFileCoverage(fileId) {
    const file = data.files[fileId];
    let totalStatements = 0;
    let coveredStatements = 0;

    file.coverage.forEach(cov => {
      if (cov > 0) totalStatements++;
      if (cov === 2) coveredStatements++;
    });

    return totalStatements === 0 ? 0 : (coveredStatements / totalStatements) * 100;
  }

  function calculateDirectoryCoverage(node) {
    if (node.type === 'file') {
      return coverageCache.get(node.fileId) || 0;
    }

    let totalCoverage = 0;
    let fileCount = 0;

    node.children?.forEach(child => {
      const childCov = calculateDirectoryCoverage(child);
      totalCoverage += childCov;
      fileCount++;
    });

    return fileCount === 0 ? 0 : totalCoverage / fileCount;
  }

  function sortTreeNodes(node, mode) {
    if (!node.children || node.children.length === 0) return node;

    // Deep copy to avoid mutating original
    const sorted = { ...node };
    sorted.children = [...node.children].map(child => sortTreeNodes(child, mode));

    // Sort children
    sorted.children.sort((a, b) => {
      // Directories always first
      if (a.type !== b.type) return a.type === 'dir' ? -1 : 1;

      if (mode === 'coverage') {
        const aCov = calculateDirectoryCoverage(a);
        const bCov = calculateDirectoryCoverage(b);
        console.log('Sorting:', a.name, '('+aCov.toFixed(1)+'%) vs', b.name, '('+bCov.toFixed(1)+'%)', '=', bCov - aCov);
        // Descending: high coverage first
        return aCov !== bCov ? bCov - aCov : a.name.localeCompare(b.name);
      }

      return a.name.localeCompare(b.name);
    });

    return sorted;
  }

  // Initialize
  function init() {
    initCoverageCache();
    loadSortPreference();
    renderSummary();
    renderTree();
    setupEventListeners();
    loadTheme();
    loadSyntaxPreference();

    // Check for deep link hash first, otherwise select first file
    if (!navigateToHash() && data.files.length > 0) {
      selectFile(0);
    }

    // Listen for hash changes (browser back/forward)
    window.addEventListener('hashchange', navigateToHash);
  }

  // Deep linking: parse URL hash
  function parseHash() {
    const hash = window.location.hash.slice(1);
    if (!hash) return null;

    const match = hash.match(/^file-(\d+)(?::line-(\d+)(?:-(\d+))?)?$/);
    if (!match) return null;

    return {
      fileId: parseInt(match[1], 10),
      lineStart: match[2] ? parseInt(match[2], 10) : null,
      lineEnd: match[3] ? parseInt(match[3], 10) : null
    };
  }

  // Deep linking: navigate to hash location
  function navigateToHash() {
    const target = parseHash();
    if (!target) return false;

    if (target.fileId < 0 || target.fileId >= data.files.length) return false;

    selectFile(target.fileId);

    if (target.lineStart) {
      requestAnimationFrame(() => {
        const lineEnd = target.lineEnd || target.lineStart;
        anchorLine = target.lineStart;
        selectedRange = { start: target.lineStart, end: lineEnd };
        selectLineRange(target.lineStart, lineEnd);
        scrollToLine(target.lineStart);
      });
    }

    return true;
  }

  // Deep linking: scroll to and highlight a line
  function scrollToLine(lineNum) {
    const lineEl = document.querySelector('.code-line[data-line="' + lineNum + '"]');
    if (!lineEl) return;

    lineEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
  }

  // Clear all selected lines
  function clearLineSelection() {
    document.querySelectorAll('.code-line.selected-line').forEach(el => {
      el.classList.remove('selected-line');
    });
  }

  // Select a range of lines (inclusive)
  function selectLineRange(start, end) {
    clearLineSelection();
    const minLine = Math.min(start, end);
    const maxLine = Math.max(start, end);
    for (let i = minLine; i <= maxLine; i++) {
      const lineEl = document.querySelector('.code-line[data-line="' + i + '"]');
      if (lineEl) {
        lineEl.classList.add('selected-line');
      }
    }
  }

  // Deep linking: update URL hash
  function updateHash(fileId, lineStart, lineEnd) {
    let hash = 'file-' + fileId;
    if (lineStart) {
      hash += ':line-' + lineStart;
      if (lineEnd && lineEnd !== lineStart) {
        // Normalise so start < end
        const minLine = Math.min(lineStart, lineEnd);
        const maxLine = Math.max(lineStart, lineEnd);
        hash = 'file-' + fileId + ':line-' + minLine + '-' + maxLine;
      }
    }
    history.replaceState(null, '', '#' + hash);
  }

  function renderSummary() {
    // Build summary safely using DOM methods
    summary.textContent = '';
    const span = document.createElement('span');
    span.className = 'percent';
    span.textContent = data.summary.percent.toFixed(1) + '%';
    summary.appendChild(span);
    summary.appendChild(document.createTextNode(
      ' coverage (' + data.summary.coveredLines + '/' + data.summary.totalLines + ' lines)'
    ));
  }

  function renderTree() {
    fileTree.textContent = '';
    // Auto-expand all top-level directories
    if (data.tree.children && data.tree.children.length > 0) {
      data.tree.children.forEach(child => {
        if (child.type === 'dir') {
          expandedDirs.add(getNodePath(child, 0));
        }
      });
    }
    const sortedTree = sortTreeNodes(data.tree, sortMode);
    renderNode(sortedTree, fileTree, 0);
  }

  function renderNode(node, container, depth) {
    if (node.name === '.' && node.type === 'dir') {
      // Root node, render children directly
      node.children.forEach(child => renderNode(child, container, depth));
      return;
    }

    const nodeEl = document.createElement('div');
    nodeEl.className = 'tree-node';
    nodeEl.dataset.name = node.name.toLowerCase();

    const item = document.createElement('div');
    item.className = 'tree-item';
    item.style.setProperty('--depth', depth);

    const icon = document.createElement('span');
    icon.className = 'icon';

    const name = document.createElement('span');
    name.className = 'name';
    name.textContent = node.name;

    if (node.type === 'dir') {
      const dirPath = getNodePath(node, depth);
      icon.textContent = expandedDirs.has(dirPath) ? '\u25BC' : '\u25B6';
      if (expandedDirs.has(dirPath)) {
        nodeEl.classList.add('expanded');
      }

      item.addEventListener('click', (e) => {
        e.stopPropagation();
        toggleDir(nodeEl, dirPath, icon);
      });

      item.appendChild(icon);
      item.appendChild(name);

      // Add coverage badge to all directories
      const badge = document.createElement('span');
      badge.className = 'coverage-badge';
      badge.textContent = calculateDirectoryCoverage(node).toFixed(1) + '%';
      item.appendChild(badge);

      nodeEl.appendChild(item);

      if (node.children && node.children.length > 0) {
        const children = document.createElement('div');
        children.className = 'tree-children';
        node.children.forEach(child => renderNode(child, children, depth + 1));
        nodeEl.appendChild(children);
      }
    } else {
      icon.textContent = '\uD83D\uDCC4';
      nodeEl.dataset.fileId = node.fileId;

      item.addEventListener('click', (e) => {
        e.stopPropagation();
        selectFile(node.fileId);
      });

      item.appendChild(icon);
      item.appendChild(name);

      // Add coverage badge to files
      const badge = document.createElement('span');
      badge.className = 'coverage-badge';
      badge.textContent = calculateDirectoryCoverage(node).toFixed(1) + '%';
      item.appendChild(badge);

      nodeEl.appendChild(item);
    }

    container.appendChild(nodeEl);
  }

  function getNodePath(node, depth) {
    return node.name + '_' + depth;
  }

  function toggleDir(nodeEl, path, icon) {
    if (nodeEl.classList.contains('expanded')) {
      nodeEl.classList.remove('expanded');
      expandedDirs.delete(path);
      icon.textContent = '\u25B6';
    } else {
      nodeEl.classList.add('expanded');
      expandedDirs.add(path);
      icon.textContent = '\u25BC';
    }
  }

  function selectFile(fileId) {
    currentFileId = fileId;
    matches = [];
    currentMatchIndex = -1;
    matchInfo.textContent = '';
    contentSearch.value = '';
    contentSearchQuery = '';
    anchorLine = null;
    selectedRange = null;

    // Update selection in tree
    document.querySelectorAll('.tree-item.selected').forEach(el => {
      el.classList.remove('selected');
    });
    const selected = document.querySelector('[data-file-id="' + fileId + '"] .tree-item');
    if (selected) {
      selected.classList.add('selected');
    }

    const file = data.files[fileId];
    if (!file) return;

    filePath.textContent = file.path;
    renderCode(file);

    // Update URL hash for deep linking
    updateHash(fileId, null);
  }

  function renderCode(file) {
    viewport.textContent = '';

    if (!file.lines || file.lines.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'empty-state';
      const iconDiv = document.createElement('div');
      iconDiv.className = 'icon';
      iconDiv.textContent = '\uD83D\uDCED';
      const textDiv = document.createElement('div');
      textDiv.textContent = 'No content';
      empty.appendChild(iconDiv);
      empty.appendChild(textDiv);
      viewport.appendChild(empty);
      return;
    }

    const container = document.createElement('div');
    container.className = 'code-container';

    file.lines.forEach((line, idx) => {
      const lineEl = document.createElement('div');
      lineEl.className = 'code-line';
      lineEl.dataset.line = idx + 1;

      const cov = file.coverage[idx];
      if (cov === 2) {
        lineEl.classList.add('covered');
      } else if (cov === 1) {
        lineEl.classList.add('uncovered');
      }

      const gutter = document.createElement('div');
      gutter.className = 'gutter';

      const lineNum = document.createElement('div');
      lineNum.className = 'line-number';
      lineNum.textContent = idx + 1;
      lineNum.title = 'Click to select line, Shift+Click for range';

      // Add click handler for line number deep linking
      const lineNumber = idx + 1;
      lineNum.addEventListener('click', (e) => {
        e.stopPropagation();

        if (e.shiftKey && anchorLine !== null) {
          // Shift-click: select range from anchor to clicked line
          const start = Math.min(anchorLine, lineNumber);
          const end = Math.max(anchorLine, lineNumber);
          selectedRange = { start: start, end: end };
          selectLineRange(start, end);
          updateHash(currentFileId, start, end);
        } else {
          // Regular click: set anchor and select single line
          anchorLine = lineNumber;
          selectedRange = { start: lineNumber, end: lineNumber };
          selectLineRange(lineNumber, lineNumber);
          updateHash(currentFileId, lineNumber, null);
        }
      });

      const content = document.createElement('div');
      content.className = 'line-content';
      content.textContent = line || ' ';

      lineEl.appendChild(gutter);
      lineEl.appendChild(lineNum);
      lineEl.appendChild(content);
      container.appendChild(lineEl);
    });

    viewport.appendChild(container);

    // Apply syntax highlighting after rendering if enabled
    if (syntaxHighlightEnabled) {
      applySyntaxHighlighting();
    }
  }

  function setupEventListeners() {
    // File search
    let searchTimeout;
    searchInput.addEventListener('input', (e) => {
      clearTimeout(searchTimeout);
      searchTimeout = setTimeout(() => {
        searchQuery = e.target.value.toLowerCase();
        filterTree();
      }, 300);
    });

    // Content search
    let contentTimeout;
    contentSearch.addEventListener('input', (e) => {
      clearTimeout(contentTimeout);
      contentTimeout = setTimeout(() => {
        contentSearchQuery = e.target.value;
        searchInFile();
      }, 300);
    });

    contentSearch.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        if (e.shiftKey) {
          goToPrevMatch();
        } else {
          goToNextMatch();
        }
      }
    });

    prevMatch.addEventListener('click', goToPrevMatch);
    nextMatch.addEventListener('click', goToNextMatch);

    // Theme toggle
    themeToggle.addEventListener('click', toggleTheme);

    // Syntax toggle
    syntaxToggle.addEventListener('click', toggleSyntax);

    // Sort controls
    const sortButtons = document.querySelectorAll('.sort-btn');
    console.log('Found', sortButtons.length, 'sort buttons');
    sortButtons.forEach(btn => {
      console.log('Attaching click handler to button:', btn.dataset.sort);
      btn.addEventListener('click', () => {
        console.log('Sort button clicked:', btn.dataset.sort);
        changeSortMode(btn.dataset.sort);
      });
    });

    // Keyboard shortcuts
    document.addEventListener('keydown', (e) => {
      if ((e.ctrlKey || e.metaKey) && e.key === 'f' && currentFileId !== null) {
        e.preventDefault();
        contentSearch.focus();
      }
      if ((e.ctrlKey || e.metaKey) && e.key === 'p') {
        e.preventDefault();
        searchInput.focus();
      }
      // Help modal
      if (e.key === '?' && !e.ctrlKey && !e.metaKey) {
        e.preventDefault();
        showHelp();
      }
      if (e.key === 'Escape') {
        // Exit search if focused
        if (document.activeElement === searchInput) {
          searchInput.value = '';
          searchQuery = '';
          filterTree();
          searchInput.blur();
          viewport.focus();
          return;
        }
        if (document.activeElement === contentSearch) {
          contentSearch.value = '';
          contentSearchQuery = '';
          matchInfo.textContent = '';
          matches = [];
          currentMatchIndex = -1;
          if (currentFileId !== null) {
            renderCode(data.files[currentFileId]);
          }
          contentSearch.blur();
          viewport.focus();
          return;
        }
        hideHelp();
      }
    });

    closeHelp.addEventListener('click', hideHelp);
    helpToggle.addEventListener('click', showHelp);
    helpModal.addEventListener('click', (e) => {
      if (e.target === helpModal) hideHelp();
    });
  }

  function filterTree() {
    const nodes = document.querySelectorAll('.tree-node');

    if (!searchQuery) {
      nodes.forEach(n => n.classList.remove('hidden'));
      return;
    }

    nodes.forEach(node => {
      const name = node.dataset.name || '';
      const fileId = node.dataset.fileId;

      if (fileId !== undefined) {
        const file = data.files[parseInt(fileId)];
        const matchesQuery = file && file.path.toLowerCase().includes(searchQuery);
        node.classList.toggle('hidden', !matchesQuery);
      } else {
        const hasVisibleChild = Array.from(node.querySelectorAll('[data-file-id]')).some(f => {
          const fid = parseInt(f.dataset.fileId);
          const file = data.files[fid];
          return file && file.path.toLowerCase().includes(searchQuery);
        });
        node.classList.toggle('hidden', !hasVisibleChild);
        if (hasVisibleChild && searchQuery) {
          node.classList.add('expanded');
          const icon = node.querySelector('.icon');
          if (icon && icon.textContent === '\u25B6') {
            icon.textContent = '\u25BC';
          }
        }
      }
    });
  }

  function searchInFile() {
    matches = [];
    currentMatchIndex = -1;

    // Re-render code to clear highlights
    if (currentFileId !== null) {
      const file = data.files[currentFileId];
      if (file) {
        renderCode(file);
      }
    }

    if (!contentSearchQuery || currentFileId === null) {
      matchInfo.textContent = '';
      return;
    }

    const file = data.files[currentFileId];
    if (!file) return;

    const query = contentSearchQuery.toLowerCase();

    file.lines.forEach((line, idx) => {
      const text = line || '';
      const lowerText = text.toLowerCase();
      let pos = 0;
      let matchIndex;

      while ((matchIndex = lowerText.indexOf(query, pos)) !== -1) {
        matches.push({ line: idx, start: matchIndex, length: query.length });
        pos = matchIndex + 1;
      }
    });

    if (matches.length > 0) {
      highlightMatches();
      currentMatchIndex = 0;
      scrollToMatch(0);
      updateMatchInfo();
    } else {
      matchInfo.textContent = 'No matches';
    }
  }

  function highlightMatches() {
    const file = data.files[currentFileId];
    if (!file) return;

    const lineEls = document.querySelectorAll('.code-line');

    // Group matches by line
    const matchesByLine = {};
    matches.forEach((m, idx) => {
      if (!matchesByLine[m.line]) matchesByLine[m.line] = [];
      matchesByLine[m.line].push({ ...m, idx });
    });

    Object.keys(matchesByLine).forEach(lineIdx => {
      const lineEl = lineEls[parseInt(lineIdx)];
      if (!lineEl) return;

      const content = lineEl.querySelector('.line-content');
      if (!content) return;

      const text = file.lines[parseInt(lineIdx)] || '';
      const lineMatches = matchesByLine[lineIdx].sort((a, b) => a.start - b.start);

      // Build content using DOM nodes for safety
      content.textContent = '';
      let lastEnd = 0;

      lineMatches.forEach(m => {
        // Text before match
        if (m.start > lastEnd) {
          content.appendChild(document.createTextNode(text.substring(lastEnd, m.start)));
        }
        // Match span
        const span = document.createElement('span');
        span.className = 'match-highlight';
        span.dataset.matchIdx = m.idx;
        span.textContent = text.substring(m.start, m.start + m.length);
        content.appendChild(span);
        lastEnd = m.start + m.length;
      });

      // Text after last match
      if (lastEnd < text.length) {
        content.appendChild(document.createTextNode(text.substring(lastEnd)));
      }

      // Handle empty line
      if (content.childNodes.length === 0) {
        content.textContent = ' ';
      }
    });
  }

  function scrollToMatch(idx) {
    document.querySelectorAll('.current-match').forEach(el => {
      el.classList.remove('current-match');
    });

    const matchEl = document.querySelector('[data-match-idx="' + idx + '"]');
    if (matchEl) {
      matchEl.classList.add('current-match');
      matchEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    }
  }

  function updateMatchInfo() {
    if (matches.length === 0) {
      matchInfo.textContent = 'No matches';
    } else {
      matchInfo.textContent = (currentMatchIndex + 1) + '/' + matches.length;
    }
  }

  function goToNextMatch() {
    if (matches.length === 0) return;
    currentMatchIndex = (currentMatchIndex + 1) % matches.length;
    scrollToMatch(currentMatchIndex);
    updateMatchInfo();
  }

  function goToPrevMatch() {
    if (matches.length === 0) return;
    currentMatchIndex = (currentMatchIndex - 1 + matches.length) % matches.length;
    scrollToMatch(currentMatchIndex);
    updateMatchInfo();
  }

  function toggleTheme() {
    const body = document.body;
    const current = body.dataset.theme;
    const next = current === 'dark' ? 'light' : 'dark';
    body.dataset.theme = next;
    localStorage.setItem('coverage-theme', next);
  }

  function loadTheme() {
    const saved = localStorage.getItem('coverage-theme');
    if (saved) {
      document.body.dataset.theme = saved;
    }
  }

  function applySyntaxHighlighting() {
    if (!syntaxHighlightEnabled || currentFileId === null) return;
    if (typeof hljs === 'undefined') return;

    const file = data.files[currentFileId];
    if (!file) return;

    const lineEls = document.querySelectorAll('.code-line');

    lineEls.forEach((lineEl, idx) => {
      const cov = file.coverage[idx];
      // Only highlight lines with no coverage info
      if (cov !== 0) return;

      const content = lineEl.querySelector('.line-content');
      if (!content || !content.textContent.trim()) return;

      const text = content.textContent;

      // Use hljs.highlight() which returns result object
      const result = hljs.highlight(text, { language: 'go' });

      // Parse the highlighted HTML safely using DOMParser
      const parser = new DOMParser();
      const doc = parser.parseFromString('<div>' + result.value + '</div>', 'text/html');
      const wrapper = doc.body.firstChild;

      // Clear and append parsed nodes
      content.textContent = '';
      while (wrapper.firstChild) {
        content.appendChild(wrapper.firstChild);
      }
    });
  }

  function toggleSyntax() {
    syntaxHighlightEnabled = !syntaxHighlightEnabled;
    syntaxToggle.classList.toggle('active', syntaxHighlightEnabled);
    localStorage.setItem('coverage-syntax', syntaxHighlightEnabled ? 'on' : 'off');

    // Re-render current file
    if (currentFileId !== null) {
      const file = data.files[currentFileId];
      if (file) {
        renderCode(file);
      }
    }
  }

  function loadSyntaxPreference() {
    const saved = localStorage.getItem('coverage-syntax');
    if (saved !== null) {
      // User preference overrides default
      syntaxHighlightEnabled = saved === 'on';
    }
    // Update button state
    syntaxToggle.classList.toggle('active', syntaxHighlightEnabled);
  }

  function changeSortMode(mode) {
    if (sortMode === mode) return;

    console.log('Changing sort mode from', sortMode, 'to', mode);
    sortMode = mode;
    localStorage.setItem('coverage-sort-mode', mode);

    // Update button states
    document.querySelectorAll('.sort-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.sort === mode);
    });

    // Re-render tree
    renderTree();
  }

  function loadSortPreference() {
    const saved = localStorage.getItem('coverage-sort-mode');
    if (saved && (saved === 'name' || saved === 'coverage')) {
      sortMode = saved;
    }

    // Update button states
    document.querySelectorAll('.sort-btn').forEach(btn => {
      btn.classList.toggle('active', btn.dataset.sort === sortMode);
    });
  }

  function showHelp() {
    helpModal.classList.remove('hidden');
  }

  function hideHelp() {
    helpModal.classList.add('hidden');
  }

  // Start the app
  init();
})();
