/**
 * Assignment data types
 */

export interface Assignment {
  ID: string;
  Parent: string;
  DecisionReason: string;
  ParentType: string;
}

export interface DayData {
  dateStr: string;
  dayOfMonth: number;
  assignmentId: string | null;
  assignmentParent: string;
  assignmentReason: string;
  isOverridden: boolean;
  cssClasses: string;
  date?: Date;
}

export interface CalendarData {
  days: DayData[];
  startDate: string;
  endDate: string;
}

export interface AssignmentDetails {
  calculation_date: string;
  parent_a_name: string;
  parent_a_total_count: number;
  parent_a_last_30_days: number;
  parent_b_name: string;
  parent_b_total_count: number;
  parent_b_last_30_days: number;
  decision_reason: string;
}
