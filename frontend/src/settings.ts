/**
 * Settings page script
 * Handles checkbox state initialization for parent unavailability
 */

import { getElements } from './utils/dom';

interface SettingsData {
  parentAUnavailable: string[];
  parentBUnavailable: string[];
}

/**
 * Initializes the settings page
 */
function initSettings(): void {
  // Get data from the template (injected via Go template)
  const settingsDataElement = document.getElementById('settings-data');
  if (!settingsDataElement) {
    console.error('Settings data element not found');
    return;
  }

  const settingsData: SettingsData = JSON.parse(settingsDataElement.textContent || '{}');
  const { parentAUnavailable, parentBUnavailable } = settingsData;

  // Set Parent A checkboxes
  const parentACheckboxes = getElements<HTMLInputElement>('input[name="parent_a_unavailable"]');
  parentACheckboxes.forEach((checkbox) => {
    if (parentAUnavailable.includes(checkbox.value)) {
      checkbox.checked = true;
    }
  });

  // Set Parent B checkboxes
  const parentBCheckboxes = getElements<HTMLInputElement>('input[name="parent_b_unavailable"]');
  parentBCheckboxes.forEach((checkbox) => {
    if (parentBUnavailable.includes(checkbox.value)) {
      checkbox.checked = true;
    }
  });
}

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', initSettings);
