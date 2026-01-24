/**
 * DOM utility functions to safely get elements
 */

/**
 * Gets an element by ID, throwing an error if not found
 */
export function getRequiredElement<T extends HTMLElement>(id: string): T {
  const element = document.getElementById(id);
  if (!element) {
    throw new Error(`Required element with id "${id}" not found`);
  }
  return element as T;
}

/**
 * Gets an element by ID, returning null if not found
 */
export function getOptionalElement<T extends HTMLElement>(id: string): T | null {
  return document.getElementById(id) as T | null;
}

/**
 * Gets all elements matching a selector
 */
export function getElements<T extends Element>(selector: string, parent: ParentNode = document): NodeListOf<T> {
  return parent.querySelectorAll<T>(selector);
}

/**
 * Creates a DOM element with text content
 */
export function createElement<K extends keyof HTMLElementTagNameMap>(
  tag: K,
  className?: string,
  textContent?: string
): HTMLElementTagNameMap[K] {
  const element = document.createElement(tag);
  if (className) {
    element.className = className;
  }
  if (textContent !== undefined) {
    element.textContent = textContent;
  }
  return element;
}
