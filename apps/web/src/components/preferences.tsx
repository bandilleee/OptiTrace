'use client';

import { createContext, useContext, useEffect, useState, type ReactNode } from 'react';
import { Moon, Sun, Pause, Play } from 'lucide-react';
import { Button } from '@/components/ui/button';

const Preferences = createContext({
  dark: true,
  paused: false,
  setDark: (_value: boolean) => {},
  setPaused: (_value: boolean) => {},
});

export function PreferencesProvider({ children }: { children: ReactNode }) {
  const [dark, setDark] = useState(true);
  const [paused, setPaused] = useState(false);
  useEffect(() => {
    try {
      setDark(localStorage.getItem('optitrace.theme') !== 'light');
      setPaused(localStorage.getItem('optitrace.motion') === 'paused');
    } catch {
      /* Storage can be disabled. */
    }
  }, []);
  function changeTheme(value: boolean) {
    setDark(value);
    document.documentElement.classList.toggle('dark', value);
    document.documentElement.dataset.theme = value ? 'dark' : 'light';
    try {
      localStorage.setItem('optitrace.theme', value ? 'dark' : 'light');
    } catch {
      /* Keep in-memory preference. */
    }
  }
  function changeMotion(value: boolean) {
    setPaused(value);
    try {
      localStorage.setItem('optitrace.motion', value ? 'paused' : 'playing');
    } catch {
      /* Keep in-memory preference. */
    }
  }
  return (
    <Preferences.Provider value={{ dark, paused, setDark: changeTheme, setPaused: changeMotion }}>
      {children}
    </Preferences.Provider>
  );
}

export function usePreferences() {
  return useContext(Preferences);
}

export function PreferenceControls() {
  const { dark, paused, setDark, setPaused } = usePreferences();
  return (
    <div className="preference-controls">
      <Button
        variant="outline"
        size="sm"
        onClick={() => setDark(!dark)}
        aria-label={dark ? 'Switch to light theme' : 'Switch to dark theme'}
      >
        {dark ? <Sun /> : <Moon />}
        <span>{dark ? 'Light theme' : 'Dark theme'}</span>
      </Button>
      <Button
        variant="outline"
        size="sm"
        onClick={() => setPaused(!paused)}
        aria-label={paused ? 'Play motion' : 'Pause motion'}
        aria-pressed={paused}
      >
        {paused ? <Play /> : <Pause />}
        <span>{paused ? 'Play motion' : 'Pause motion'}</span>
      </Button>
    </div>
  );
}
