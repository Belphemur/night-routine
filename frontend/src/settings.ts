/**
 * Settings page script
 * Handles checkbox state initialization for parent unavailability
 */

import { getElements } from './utils/dom';

interface SettingsData {
  parentAUnavailable: string[];
  parentBUnavailable: string[];
}

// Extend Window interface for type safety
declare global {
  interface Window {
    SETTINGS_DATA?: SettingsData;
  }
}

/**
 * Initializes the settings page
 */
function initSettings(): void {
  // Get data from the window object (injected via template)
  const settingsData = window.SETTINGS_DATA;
  if (!settingsData) {
    console.error('Settings data not found on window object');
    return;
  }

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
