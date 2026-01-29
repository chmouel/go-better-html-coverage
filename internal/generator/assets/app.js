(function() {
  'use strict';

  const data = window.COVERAGE_DATA;

  // State
  let currentFileId = null;
  let searchQuery = '';
  let contentSearchQuery = '';
  let matches = [];
  let currentMatchIndex = -1;
  let expandedDirs = new Set();

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

  // Initialize
  function init() {
    renderSummary();
    renderTree();
    setupEventListeners();
    loadTheme();

    // Select first file if available
    if (data.files.length > 0) {
      selectFile(0);
    }
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
    renderNode(data.tree, fileTree, 0);
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

      const content = document.createElement('div');
      content.className = 'line-content';
      content.textContent = line || ' ';

      lineEl.appendChild(gutter);
      lineEl.appendChild(lineNum);
      lineEl.appendChild(content);
      container.appendChild(lineEl);
    });

    viewport.appendChild(container);
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

  // Start the app
  init();
})();
