/**
 * Mobile weekly calendar management
 */

import { getLocalDateString, parseDateString } from '../utils/date';
import { getOptionalElement, getRequiredElement, createElement } from '../utils/dom';
import { showUnlockModal } from './unlockModal';
import { showDetailsModal } from './detailsModal';
import type { CalendarData, DayData } from './types';

/**
 * Initializes the mobile calendar with weekly navigation
 */
export function initMobileCalendar(): void {
  const row1 = getOptionalElement('mobile-assignment-calendar-row1');
  const row2 = getOptionalElement('mobile-assignment-calendar-row2');
  
  if (!row1 || !row2) return;

  const calendarDataElement = document.getElementById('calendar-data');
  if (!calendarDataElement) {
    console.error('Calendar data element not found');
    return;
  }

  const mobileData: CalendarData = JSON.parse(calendarDataElement.textContent || '{}');
  const allDays = mobileData.days.map(day => ({
    ...day,
    date: parseDateString(day.dateStr)
  }));

  const allDaysMap = new Map(allDays.map(day => [day.dateStr, day]));

  let currentWeekOffset = 0;

  const prevBtn = getRequiredElement<HTMLButtonElement>('prev-week-btn');
  const nextBtn = getRequiredElement<HTMLButtonElement>('next-week-btn');
  const currentBtn = getRequiredElement<HTMLButtonElement>('current-week-btn');

  /**
   * Gets Monday of a given week
   */
  function getMondayOfWeek(date: Date): Date {
    const d = new Date(date.getTime());
    const day = d.getDay();
    const diff = d.getDate() - day + (day === 0 ? -6 : 1);
    d.setDate(diff);
    return d;
  }

  /**
   * Formats week label
   */
  function formatWeekLabel(mondayDate: Date): string {
    const sunday = new Date(mondayDate.getTime());
    sunday.setDate(sunday.getDate() + 6);

    const options: Intl.DateTimeFormatOptions = { month: 'long', day: 'numeric' };
    const mondayStr = mondayDate.toLocaleDateString('en-US', options);
    const sundayStr = sunday.toLocaleDateString('en-US', { ...options, year: 'numeric' });

    if (mondayDate.getMonth() === sunday.getMonth()) {
      return `Week of ${mondayStr} - ${sunday.getDate()}, ${sunday.getFullYear()}`;
    }
    return `Week of ${mondayStr} - ${sundayStr}`;
  }

  /**
   * Creates a day cell for the mobile calendar
   */
  function createDayCell(day: DayData, todayString: string): HTMLTableCellElement {
    const td = document.createElement('td');
    // Combine classes - ensure all required Tailwind classes are present
    // Tailwind CSS classes used dynamically - DO NOT REMOVE
    // Classes: h-24 p-2 text-xs text-lg block font-bold mb-1 font-semibold text-slate-500 mt-1
    td.className = [day.cssClasses, 'h-24', 'p-2', 'text-xs'].filter(Boolean).join(' ');
    td.setAttribute('data-date', day.dateStr);
    
    if (day.assignmentId) {
      td.setAttribute('data-assignment-id', day.assignmentId);
      td.style.cursor = 'pointer';
    }

    // Build aria-label
    const dateObj = parseDateString(day.dateStr);
    let ariaLabel = dateObj.toLocaleDateString('en-US', {
      weekday: 'long',
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
    
    if (day.assignmentParent) {
      ariaLabel += ` - ${day.assignmentParent} assigned`;
      if (day.isOverridden) {
        ariaLabel += ' - Locked (manually overridden)';
      }
    }
    td.setAttribute('aria-label', ariaLabel);

    // Highlight today
    if (day.dateStr === todayString) {
      td.classList.add('today-cell');
    }

    // Build content
    td.appendChild(createElement('span', 'block text-lg font-bold mb-1', day.dayOfMonth.toString()));
    
    if (day.assignmentParent) {
      td.appendChild(createElement('span', 'block text-xs font-semibold', day.assignmentParent));
    }
    
    if (day.assignmentReason) {
      td.appendChild(createElement('span', 'block text-xs text-slate-500 mt-1', day.assignmentReason));
    }

    // Add click handler
    if (day.assignmentId) {
      td.addEventListener('click', (e) => {
        e.stopPropagation();
        if (day.isOverridden) {
          showUnlockModal(day.assignmentId!);
        } else {
          showDetailsModal(day.assignmentId!);
        }
      });
    }

    return td;
  }

  /**
   * Renders the current week
   */
  function renderWeek(): void {
    const today = new Date();
    const todayString = getLocalDateString(today);
    const mondayOfTargetWeek = getMondayOfWeek(today);
    mondayOfTargetWeek.setDate(mondayOfTargetWeek.getDate() + (currentWeekOffset * 7));

    // Update week label
    const weekLabel = formatWeekLabel(mondayOfTargetWeek);
    const labelElement = getRequiredElement('mobile-week-label');
    labelElement.textContent = weekLabel;

    // Get the 7 days for this week
    const weekDays: DayData[] = [];
    for (let i = 0; i < 7; i++) {
      const currentDate = new Date(mondayOfTargetWeek);
      currentDate.setDate(currentDate.getDate() + i);
      const dateStr = getLocalDateString(currentDate);

      const dayData = allDaysMap.get(dateStr);
      const defaultClasses = 'border border-slate-200 p-2 text-center align-top h-20 relative bg-white';

      weekDays.push(dayData || {
        date: currentDate,
        dateStr,
        dayOfMonth: currentDate.getDate(),
        assignmentId: null,
        assignmentParent: '',
        assignmentReason: '',
        isOverridden: false,
        cssClasses: defaultClasses
      });
    }

    // Render first row: Mon-Thu (days 0-3)
    const tbody1 = getRequiredElement<HTMLTableSectionElement>('mobile-calendar-body-row1');
    tbody1.replaceChildren();
    const row1Element = document.createElement('tr');
    for (let i = 0; i < 4; i++) {
      row1Element.appendChild(createDayCell(weekDays[i], todayString));
    }
    tbody1.appendChild(row1Element);

    // Render second row: Fri-Sun (days 4-6)
    const tbody2 = getRequiredElement<HTMLTableSectionElement>('mobile-calendar-body-row2');
    tbody2.replaceChildren();
    const row2Element = document.createElement('tr');
    for (let i = 4; i < 7; i++) {
      row2Element.appendChild(createDayCell(weekDays[i], todayString));
    }
    tbody2.appendChild(row2Element);

    // Update button states
    updateButtonState(prevBtn, canNavigatePrevious(mondayOfTargetWeek));
    updateButtonState(nextBtn, canNavigateNext(mondayOfTargetWeek));
  }

  /**
   * Checks if previous week navigation is available
   */
  function canNavigatePrevious(mondayOfTargetWeek: Date): boolean {
    const prevWeekMonday = new Date(mondayOfTargetWeek);
    prevWeekMonday.setDate(prevWeekMonday.getDate() - 7);
    const prevWeekMondayStr = getLocalDateString(prevWeekMonday);
    return mobileData.startDate ? prevWeekMondayStr >= mobileData.startDate : false;
  }

  /**
   * Checks if next week navigation is available
   */
  function canNavigateNext(mondayOfTargetWeek: Date): boolean {
    const nextWeekMonday = new Date(mondayOfTargetWeek);
    nextWeekMonday.setDate(nextWeekMonday.getDate() + 7);
    const nextWeekSunday = new Date(nextWeekMonday);
    nextWeekSunday.setDate(nextWeekSunday.getDate() + 6);
    const nextWeekSundayStr = getLocalDateString(nextWeekSunday);
    return mobileData.endDate ? nextWeekSundayStr <= mobileData.endDate : false;
  }

  /**
   * Updates button state
   */
  function updateButtonState(button: HTMLButtonElement, enabled: boolean): void {
    button.disabled = !enabled;
    button.classList.toggle('opacity-50', !enabled);
    button.classList.toggle('cursor-not-allowed', !enabled);
    button.classList.toggle('hover:bg-indigo-600', enabled);
    button.classList.toggle('hover:shadow-lg', enabled);
  }

  // Navigation handlers
  prevBtn.addEventListener('click', () => {
    currentWeekOffset--;
    renderWeek();
  });

  nextBtn.addEventListener('click', () => {
    currentWeekOffset++;
    renderWeek();
  });

  currentBtn.addEventListener('click', () => {
    currentWeekOffset = 0;
    renderWeek();
  });

  // Initial render
  renderWeek();
}
