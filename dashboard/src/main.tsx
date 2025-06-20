import React from 'react'
import { createRoot } from 'react-dom/client'
import './index.css'
import App from './App.tsx'

console.log('main.tsx executing...');
const rootElement = document.getElementById('root');
console.log('Root element found:', rootElement);

if (!rootElement) {
  console.error('Root element not found!');
  // Create a fallback element
  const fallback = document.createElement('div');
  fallback.style.padding = '20px';
  fallback.style.backgroundColor = 'lightyellow';
  fallback.style.border = '2px solid red';
  fallback.innerHTML = '<h1>Error: Root element not found</h1>';
  document.body.appendChild(fallback);
} else {
  try {
    console.log('Creating root...');
    const root = createRoot(rootElement);
    console.log('Root created successfully');

    console.log('Rendering app...');
    root.render(
      <React.StrictMode>
        <App />
      </React.StrictMode>
    );
    console.log('Render called successfully');
  } catch (error) {
    console.error('Error rendering the app:', error);
    // Display error on page
    rootElement.innerHTML = `
      <div style="padding: 20px; background-color: lightyellow; border: 2px solid red;">
        <h1>React Rendering Error</h1>
        <pre>${error instanceof Error ? error.message : String(error)}</pre>
      </div>
    `;
  }
}
