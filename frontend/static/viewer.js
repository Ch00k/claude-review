// Claude Review - Markdown Viewer with Custom Text Selection and Commenting

(function () {
    'use strict';

    let currentSelection = null;
    let commentButton = null;
    let commentPopup = null;
    let commentPanel = null;

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }

    function init() {
        initTextSelection();
        createCommentButton();
        createCommentPopup();
        createCommentPanel();
        loadExistingComments();
        // setupSSE(); // Disabled for now
    }

    function initTextSelection() {
        const container = document.getElementById('markdown-content');
        if (!container) {
            console.error('Markdown content container not found');
            return;
        }

        // Listen for text selection
        document.addEventListener('mouseup', handleTextSelection);
    }

    function handleTextSelection(event) {
        const selection = window.getSelection();
        const selectedText = selection.toString().trim();

        // Don't interfere if clicking inside the popup or button
        if (commentPopup && commentPopup.contains(event.target)) {
            return;
        }
        if (commentButton && commentButton.contains(event.target)) {
            return;
        }

        // Hide button if no text selected
        if (!selectedText) {
            hideCommentButton();
            hideCommentPopup();
            return;
        }

        // Check if selection is within markdown-content
        const container = document.getElementById('markdown-content');
        if (!container.contains(selection.anchorNode)) {
            return;
        }

        // Check if selection is within an existing comment highlight
        const range = selection.getRangeAt(0);
        let node = range.commonAncestorContainer;
        if (node.nodeType === Node.TEXT_NODE) {
            node = node.parentElement;
        }
        if (node && node.closest('.comment-highlight')) {
            // Already highlighted - don't show button
            hideCommentButton();
            return;
        }

        // Store selection info
        const { lineStart, lineEnd } = extractLineNumbersFromRange(range);

        currentSelection = {
            text: selectedText,
            range: range.cloneRange(),
            lineStart,
            lineEnd,
        };

        // Get selection bounding rect to position button at top-right
        const rect = range.getBoundingClientRect();
        showCommentButton(rect.right + window.scrollX, rect.top + window.scrollY);
    }

    function createCommentButton() {
        commentButton = document.createElement('div');
        commentButton.id = 'comment-button';
        commentButton.style.display = 'none';
        commentButton.innerHTML = `
            <button class="comment-add-btn">
                <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="currentColor" class="comment-icon">
                    <path fill-rule="evenodd" d="M4.848 2.771A49.144 49.144 0 0 1 12 2.25c2.43 0 4.817.178 7.152.52 1.978.292 3.348 2.024 3.348 3.97v6.02c0 1.946-1.37 3.678-3.348 3.97-1.94.284-3.916.455-5.922.505a.39.39 0 0 0-.266.112L8.78 21.53A.75.75 0 0 1 7.5 21v-3.955a48.842 48.842 0 0 1-2.652-.316c-1.978-.29-3.348-2.024-3.348-3.97V6.741c0-1.946 1.37-3.68 3.348-3.97Z" clip-rule="evenodd" />
                </svg>
            </button>
        `;
        document.body.appendChild(commentButton);

        // Click handler
        commentButton.querySelector('.comment-add-btn').addEventListener('click', (e) => {
            e.preventDefault();
            e.stopPropagation();
            const rect = commentButton.getBoundingClientRect();
            // Add scroll offset to get page coordinates
            const x = rect.left + window.scrollX;
            const y = rect.bottom + window.scrollY;
            showCommentPopup(x, y);
            hideCommentButton();
        });

        // Close button when clicking outside
        document.addEventListener('mousedown', (e) => {
            if (commentButton.style.display === 'block' && !commentButton.contains(e.target)) {
                // Check if there's still a text selection
                const selection = window.getSelection();
                if (!selection.toString().trim()) {
                    hideCommentButton();
                }
            }
        });
    }

    function showCommentButton(x, y) {
        commentButton.style.display = 'block';
        // Position so bottom-left corner of button touches top-right corner of selection
        commentButton.style.left = x + 'px';
        // Offset by button height to align bottom of button with top of selection
        const buttonHeight = 40; // 32px icon + 2*4px padding
        commentButton.style.top = (y - buttonHeight) + 'px';
    }

    function hideCommentButton() {
        if (commentButton) {
            commentButton.style.display = 'none';
        }
    }

    function createCommentPanel() {
        commentPanel = document.createElement('div');
        commentPanel.id = 'comment-panel';
        commentPanel.className = 'expanded';
        commentPanel.innerHTML = `
            <div class="comment-panel-header">
                <h3>Comments</h3>
                <span class="comment-count">0</span>
            </div>
            <div class="comment-panel-list"></div>
        `;
        document.body.appendChild(commentPanel);

        // Click on header to toggle collapse/expand
        commentPanel.querySelector('.comment-panel-header').addEventListener('click', () => {
            commentPanel.classList.toggle('expanded');
            commentPanel.classList.toggle('collapsed');
        });
    }

    function updateCommentPanel() {
        if (!commentPanel) return;

        const listContainer = commentPanel.querySelector('.comment-panel-list');
        const countElement = commentPanel.querySelector('.comment-count');

        // Get all comment highlights from the DOM
        const highlights = document.querySelectorAll('.comment-highlight');

        countElement.textContent = highlights.length;

        // Clear existing list
        listContainer.innerHTML = '';

        // Add each comment to the panel
        highlights.forEach((highlight, index) => {
            const commentId = highlight.dataset.commentId;
            const commentText = highlight.dataset.commentText;
            const selectedText = highlight.dataset.selectedText;
            const lineStart = highlight.dataset.lineStart;
            const lineEnd = highlight.dataset.lineEnd;

            const lineRange = lineStart === lineEnd ? `L${lineStart}` : `L${lineStart}-${lineEnd}`;

            const item = document.createElement('div');
            item.className = 'comment-panel-item';
            item.innerHTML = `
                <div class="comment-panel-item-number">${lineRange}</div>
                <div class="comment-panel-item-content">
                    <div class="comment-panel-item-text">"${selectedText}"</div>
                    <div class="comment-panel-item-comment">${commentText}</div>
                </div>
            `;

            // Click to scroll to comment
            item.addEventListener('click', () => {
                highlight.scrollIntoView({ behavior: 'smooth', block: 'center' });
                // Briefly highlight the comment
                highlight.style.backgroundColor = '#ffeb99';
                setTimeout(() => {
                    highlight.style.backgroundColor = '#fff8c5';
                }, 1000);
            });

            listContainer.appendChild(item);
        });
    }

    function createCommentPopup() {
        commentPopup = document.createElement('div');
        commentPopup.id = 'comment-popup';
        commentPopup.style.display = 'none';
        commentPopup.innerHTML = `
            <div class="comment-popup-content">
                <textarea id="comment-text" placeholder="Add your comment..." rows="4"></textarea>
                <div class="comment-popup-buttons">
                    <button id="comment-save" class="comment-btn comment-btn-primary">Add</button>
                    <button id="comment-delete" class="comment-btn comment-btn-danger" style="display: none;">Delete</button>
                    <button id="comment-cancel" class="comment-btn">Cancel</button>
                </div>
            </div>
        `;
        document.body.appendChild(commentPopup);

        // Event listeners will be set dynamically based on mode

        // Close popup when clicking outside (but not on text selection)
        document.addEventListener('mousedown', (e) => {
            if (commentPopup.style.display === 'block' && !commentPopup.contains(e.target)) {
                // Check if there's a text selection - if so, don't close
                const selection = window.getSelection();
                if (!selection.toString().trim()) {
                    hideCommentPopup();
                }
            }
        });
    }

    function showCommentPopup(x, y) {
        // Setup for adding new comment
        const saveBtn = document.getElementById('comment-save');
        const deleteBtn = document.getElementById('comment-delete');
        const cancelBtn = document.getElementById('comment-cancel');

        saveBtn.textContent = 'Add';
        deleteBtn.style.display = 'none';

        // Remove old listeners
        saveBtn.replaceWith(saveBtn.cloneNode(true));
        cancelBtn.replaceWith(cancelBtn.cloneNode(true));

        // Add new listeners
        document.getElementById('comment-save').addEventListener('click', handleAddComment);
        document.getElementById('comment-cancel').addEventListener('click', hideCommentPopup);

        commentPopup.style.display = 'block';
        commentPopup.style.left = x + 'px';
        commentPopup.style.top = y + 10 + 'px';

        // Focus textarea without scrolling
        const textarea = document.getElementById('comment-text');
        textarea.value = '';
        textarea.focus({ preventScroll: true });
    }

    function showEditCommentPopup(comment, highlightElement, x, y) {
        // Setup for editing existing comment
        const saveBtn = document.getElementById('comment-save');
        const deleteBtn = document.getElementById('comment-delete');
        const cancelBtn = document.getElementById('comment-cancel');

        saveBtn.textContent = 'Save';
        deleteBtn.style.display = 'inline-block';

        // Remove old listeners
        saveBtn.replaceWith(saveBtn.cloneNode(true));
        deleteBtn.replaceWith(deleteBtn.cloneNode(true));
        cancelBtn.replaceWith(cancelBtn.cloneNode(true));

        // Add new listeners
        document
            .getElementById('comment-save')
            .addEventListener('click', () => handleUpdateComment(comment, highlightElement));
        document
            .getElementById('comment-delete')
            .addEventListener('click', () => handleDeleteComment(comment, highlightElement));
        document.getElementById('comment-cancel').addEventListener('click', hideCommentPopup);

        commentPopup.style.display = 'block';
        commentPopup.style.left = x + 'px';
        commentPopup.style.top = y + 10 + 'px';

        // Pre-fill textarea with existing comment
        const textarea = document.getElementById('comment-text');
        textarea.value = comment.comment_text;
        textarea.focus({ preventScroll: true });
    }

    function hideCommentPopup(clearSelection = true) {
        if (commentPopup) {
            commentPopup.style.display = 'none';
        }
        currentSelection = null;
        if (clearSelection) {
            window.getSelection().removeAllRanges();
        }
    }

    /**
     * Extract line numbers from a DOM Range by finding parent elements
     * with data-line-start and data-line-end attributes
     */
    function extractLineNumbersFromRange(range) {
        let lineStart = null;
        let lineEnd = null;

        // Find all block elements with line numbers that intersect with the range
        const content = document.getElementById('markdown-content');
        const blockElements = content.querySelectorAll('[data-line-start]');

        for (const element of blockElements) {
            // Check if this element contains any part of the selection
            if (range.intersectsNode(element)) {
                const start = parseInt(element.getAttribute('data-line-start'), 10);
                const end = parseInt(element.getAttribute('data-line-end'), 10);

                if (lineStart === null || start < lineStart) {
                    lineStart = start;
                }
                if (lineEnd === null || end > lineEnd) {
                    lineEnd = end;
                }
            }
        }

        return { lineStart, lineEnd };
    }

    /**
     * Handle adding a new comment
     */
    async function handleAddComment() {
        if (!currentSelection) {
            return;
        }

        const commentText = document.getElementById('comment-text').value.trim();
        if (!commentText) {
            alert('Please enter a comment');
            return;
        }

        // Prepare comment payload
        const payload = {
            project_directory: projectDir,
            file_path: filePath,
            line_start: currentSelection.lineStart,
            line_end: currentSelection.lineEnd,
            selected_text: currentSelection.text,
            comment_text: commentText,
        };

        try {
            const response = await fetch('/api/comments', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(payload),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const savedComment = await response.json();

            // Find and highlight the text in the document
            highlightCommentByText(savedComment);

            // Update comment panel
            updateCommentPanel();

            // Hide popup and clear selection
            hideCommentPopup(true);
        } catch (error) {
            console.error('Failed to save comment:', error);
            alert('Failed to save comment. Please try again.');
        }
    }

    /**
     * Handle updating an existing comment
     */
    async function handleUpdateComment(comment, highlightElement) {
        const commentText = document.getElementById('comment-text').value.trim();
        if (!commentText) {
            alert('Please enter a comment');
            return;
        }

        try {
            const response = await fetch(`/api/comments/${comment.id}`, {
                method: 'PATCH',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({
                    comment_text: commentText,
                }),
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            const updatedComment = await response.json();

            // Update the highlight element
            highlightElement.dataset.commentText = commentText;
            highlightElement.title = commentText;

            // Update comment panel
            updateCommentPanel();

            // Hide popup
            hideCommentPopup();
        } catch (error) {
            console.error('Failed to update comment:', error);
            alert('Failed to update comment. Please try again.');
        }
    }

    /**
     * Handle deleting a comment
     */
    async function handleDeleteComment(comment, highlightElement) {
        if (!confirm('Are you sure you want to delete this comment?')) {
            return;
        }

        try {
            const response = await fetch(`/api/comments/${comment.id}`, {
                method: 'DELETE',
            });

            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }

            // Remove the highlight from the DOM
            const parent = highlightElement.parentNode;
            while (highlightElement.firstChild) {
                parent.insertBefore(highlightElement.firstChild, highlightElement);
            }
            parent.removeChild(highlightElement);

            // Update comment panel
            updateCommentPanel();

            // Hide popup
            hideCommentPopup();
        } catch (error) {
            console.error('Failed to delete comment:', error);
            alert('Failed to delete comment. Please try again.');
        }
    }

    /**
     * Highlight a comment in the document
     */
    function highlightComment(range, comment) {
        // Create a span to wrap the selected text
        const highlight = document.createElement('span');
        highlight.className = 'comment-highlight';
        highlight.dataset.commentId = comment.id;
        highlight.dataset.commentText = comment.comment_text;
        highlight.dataset.selectedText = comment.selected_text;
        highlight.dataset.lineStart = comment.line_start;
        highlight.dataset.lineEnd = comment.line_end;
        highlight.title = comment.comment_text;

        // Click handler to edit comment
        highlight.addEventListener('click', (e) => {
            e.stopPropagation();
            showEditCommentPopup(comment, highlight, e.pageX, e.pageY);
        });

        try {
            range.surroundContents(highlight);
        } catch (e) {
            // If surroundContents fails (e.g., range spans multiple elements),
            // just skip highlighting for now
            console.error('Could not highlight comment:', e);
        }
    }

    /**
     * Load existing comments from backend and display them
     */
    function loadExistingComments() {
        if (typeof comments === 'undefined') {
            console.warn('No comments data found in page');
            return;
        }

        if (!comments || comments.length === 0) {
            return;
        }

        // Highlight each comment by finding its text in the document
        comments.forEach((comment) => {
            highlightExistingComment(comment);
        });

        // Update comment panel after loading all comments
        updateCommentPanel();
    }

    /**
     * Highlight a comment by finding its text in the document within the specified line range
     */
    function highlightCommentByText(comment) {
        const content = document.getElementById('markdown-content');
        const text = comment.selected_text;

        // Find the block element(s) that contain the line range
        const blockElements = content.querySelectorAll('[data-line-start]');
        const relevantBlocks = [];

        for (const element of blockElements) {
            const lineStart = parseInt(element.getAttribute('data-line-start'), 10);
            const lineEnd = parseInt(element.getAttribute('data-line-end'), 10);

            // Check if this block overlaps with the comment's line range
            if (lineStart <= comment.line_end && lineEnd >= comment.line_start) {
                relevantBlocks.push(element);
            }
        }

        // Search for text only within the relevant blocks
        for (const block of relevantBlocks) {
            const walker = document.createTreeWalker(block, NodeFilter.SHOW_TEXT, null, false);

            let node;
            while ((node = walker.nextNode())) {
                const index = node.textContent.indexOf(text);
                if (index !== -1) {
                    const range = document.createRange();
                    range.setStart(node, index);
                    range.setEnd(node, index + text.length);

                    highlightComment(range, comment);
                    return; // Found and highlighted, we're done
                }
            }
        }
    }

    /**
     * Highlight an existing comment by finding its text in the document
     */
    function highlightExistingComment(comment) {
        highlightCommentByText(comment);
    }

    /**
     * Setup Server-Sent Events for live updates
     */
    function setupSSE() {
        const params = new URLSearchParams({
            project_directory: projectDir,
            file_path: filePath,
        });

        const eventSource = new EventSource(`/events?${params}`);

        eventSource.addEventListener('file_updated', (event) => {
            console.log('File updated event received:', event.data);
            // Reload the page to show updated content
            window.location.reload();
        });

        eventSource.addEventListener('comments_resolved', (event) => {
            console.log('Comments resolved event received:', event.data);
            const data = JSON.parse(event.data);
            // Reload to show resolved comments removed
            window.location.reload();
        });

        eventSource.onerror = (error) => {
            console.error('SSE error:', error);
            eventSource.close();

            // Attempt to reconnect after 5 seconds
            setTimeout(setupSSE, 5000);
        };
    }
})();
