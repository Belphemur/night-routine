/**
 * Assignment details modal management
 */

import { getRequiredElement } from '../utils/dom';
import { showModal, hideModal, setupBackdropClose, type ModalElements } from '../utils/modal';
import { createElement } from '../utils/dom';
import type { AssignmentDetails } from './types';

/**
 * Initializes the details modal
 */
export function initDetailsModal(): void {
  const modal = getRequiredElement('details-modal');
  const backdrop = getRequiredElement('details-modal-backdrop');
  const panel = getRequiredElement('details-modal-panel');
  const closeBtn = getRequiredElement('details-modal-close');

  closeBtn.addEventListener('click', () => hideDetailsModal({ modal, backdrop, panel }));
  setupBackdropClose(modal, panel, () => hideDetailsModal({ modal, backdrop, panel }));
}

/**
 * Shows the details modal and fetches assignment details
 */
export function showDetailsModal(assignmentId: string): void {
  const modal = getRequiredElement('details-modal');
  const backdrop = getRequiredElement('details-modal-backdrop');
  const panel = getRequiredElement('details-modal-panel');
  const content = getRequiredElement('details-modal-content');
  const closeBtn = getRequiredElement('details-modal-close');

  content.innerHTML = '<p class="text-sm text-gray-500">Loading...</p>';
  showModal({ modal, backdrop, panel }, closeBtn);

  // Fetch assignment details
  void fetchAssignmentDetails(assignmentId, content);
}

/**
 * Hides the details modal
 */
function hideDetailsModal(elements: ModalElements): void {
  hideModal(elements);
}

/**
 * Fetches and displays assignment details
 */
async function fetchAssignmentDetails(assignmentId: string, content: HTMLElement): Promise<void> {
  try {
    const response = await fetch(`/api/assignment-details?assignment_id=${assignmentId}`);
    
    if (!response.ok) {
      throw new Error('Failed to fetch assignment details');
    }

    const data = await response.json() as AssignmentDetails;
    content.replaceChildren(buildDetailsContent(data));
  } catch (error) {
    console.error('Error fetching assignment details:', error);
    const errorContainer = createElement('div', 'bg-red-50 rounded-lg p-3');
    const errorText = createElement('p', 'text-sm text-red-700', 
      'Failed to load assignment details. This assignment may not have detailed information available.');
    errorContainer.appendChild(errorText);
    content.replaceChildren(errorContainer);
  }
}

/**
 * Builds the details content DOM
 */
function buildDetailsContent(data: AssignmentDetails): HTMLElement {
  const container = createElement('div', 'space-y-3');

  // Calculation date section
  const dateSection = createElement('div', 'bg-gray-50 rounded-lg p-3');
  dateSection.appendChild(createElement('p', 'text-xs text-gray-500 uppercase tracking-wide font-semibold mb-2', 'Calculation Date'));
  dateSection.appendChild(createElement('p', 'text-sm font-medium text-gray-900', data.calculation_date));
  container.appendChild(dateSection);

  // Parents grid
  const grid = createElement('div', 'grid grid-cols-2 gap-3');
  
  // Parent A section
  grid.appendChild(buildParentSection(
    data.parent_a_name,
    data.parent_a_total_count,
    data.parent_a_last_30_days,
    'blue'
  ));

  // Parent B section
  grid.appendChild(buildParentSection(
    data.parent_b_name,
    data.parent_b_total_count,
    data.parent_b_last_30_days,
    'orange'
  ));

  container.appendChild(grid);

  // Decision reason section
  container.appendChild(buildDecisionReasonSection(data.decision_reason));

  // Explanation section
  container.appendChild(buildExplanationSection());

  return container;
}

/**
 * Builds a parent stats section
 */
function buildParentSection(
  name: string,
  totalCount: number,
  last30Days: number,
  color: 'blue' | 'orange'
): HTMLElement {
  const section = createElement('div', `bg-${color}-50 rounded-lg p-3`);
  section.appendChild(createElement('p', `text-xs text-${color}-700 uppercase tracking-wide font-semibold mb-2`, name));

  const stats = createElement('div', 'space-y-1');
  
  const totalP = createElement('p', 'text-sm text-gray-700');
  totalP.appendChild(createElement('span', 'font-medium', 'Total:'));
  totalP.appendChild(document.createTextNode(` ${totalCount}`));
  stats.appendChild(totalP);

  const last30P = createElement('p', 'text-sm text-gray-700');
  last30P.appendChild(createElement('span', 'font-medium', 'Last 30 days:'));
  last30P.appendChild(document.createTextNode(` ${last30Days}`));
  stats.appendChild(last30P);

  section.appendChild(stats);
  return section;
}

/**
 * Builds the decision reason section
 */
function buildDecisionReasonSection(decisionReason: string): HTMLElement {
  const section = createElement('div', 'bg-purple-50 rounded-lg p-3');
  section.appendChild(createElement('p', 'text-xs text-purple-700 uppercase tracking-wide font-semibold mb-2', 'Decision Reason'));
  section.appendChild(createElement('p', 'text-sm font-bold text-purple-900 mb-2', decisionReason));

  const explanations: Record<string, string> = {
    'Unavailability': 'One parent was unavailable on this day based on configured schedule constraints.',
    'Total Count': 'This parent had fewer total assignments overall, helping maintain long-term balance.',
    'Recent Count': 'This parent had fewer assignments in the last 30 days, ensuring fair recent distribution.',
    'Consecutive Limit': 'Prevents one parent from having too many consecutive night assignments (limit: 2).',
    'Alternating': 'Both parents had equal counts, so the algorithm maintained an alternating pattern.',
    'Override': 'This assignment was manually changed in Google Calendar by a user.'
  };

  const explanation = explanations[decisionReason] || 'Assignment made by the fairness algorithm.';
  section.appendChild(createElement('p', 'text-xs text-gray-600 italic', explanation));

  return section;
}

/**
 * Builds the algorithm explanation section
 */
function buildExplanationSection(): HTMLElement {
  const section = createElement('div', 'bg-indigo-50 rounded-lg p-3');
  section.appendChild(createElement('p', 'text-xs text-indigo-700 uppercase tracking-wide font-semibold mb-1', 'How the algorithm works'));
  section.appendChild(createElement('p', 'text-sm text-gray-700', 
    'The fairness algorithm evaluates multiple criteria in priority order: (1) Parent availability, (2) Total assignment counts, (3) Recent counts (last 30 days), (4) Consecutive limit (max 2), (5) Alternating pattern.'));
  
  return section;
}
