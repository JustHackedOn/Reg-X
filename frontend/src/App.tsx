import React, { useState, useEffect } from 'react';

// Wails auto-generated bindings are accessed via the window object
// before running `wails generate module` or during `wails dev`.
const api = (window as any).go?.main?.App;

type Tab = 'Encrypt' | 'Decrypt' | 'Settings';

interface ProgressEvent {
  current: number;
  total: number;
  filename: string;
  percent: number;
  speed: string;
  eta: string;
}

interface OperationResult {
  success: number;
  failed: number;
  total: number;
  errors: string[];
  elapsedTime: string;
}

interface Settings {
  extension: string;
  theme: string;
  deleteOriginals: boolean;
  outputFolder: string;
  outputMode: string;
}

export default function App() {
  const [activeTab, setActiveTab] = useState<Tab>('Encrypt');
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  
  // Settings State
  const [settings, setSettings] = useState<Settings>({
    extension: '.pse',
    theme: 'dark',
    deleteOriginals: false,
    outputFolder: '',
    outputMode: 'alongside'
  });

  // Operation State
  const [selectedPaths, setSelectedPaths] = useState<string[]>([]);
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [isProcessing, setIsProcessing] = useState(false);
  
  // Progress State
  const [progress, setProgress] = useState<ProgressEvent | null>(null);
  const [result, setResult] = useState<OperationResult | null>(null);

  // Track how the user selected paths (file dialog vs folder dialog)
  const [selectionMode, setSelectionMode] = useState<'files' | 'folder'>('files');

  // Load settings on mount
  useEffect(() => {
    if (api) {
      api.GetSettings().then((s: Settings) => {
        setSettings(s);
        setTheme(s.theme as 'dark' | 'light');
      });
    }
    
    // Listen for progress events from the Go backend
    if ((window as any).runtime?.EventsOn) {
      (window as any).runtime.EventsOn('progress', (p: ProgressEvent) => {
        setProgress(p);
      });
    }
  }, []);

  // Sync theme with HTML class for Tailwind dark mode
  useEffect(() => {
    if (theme === 'dark') {
      document.documentElement.classList.add('dark');
    } else {
      document.documentElement.classList.remove('dark');
    }
  }, [theme]);

  const handleSaveSettings = async (newSettings: Settings) => {
    setSettings(newSettings);
    setTheme(newSettings.theme as 'dark' | 'light');
    if (api) {
      await api.SaveSettings(newSettings);
    }
  };

  const clearState = () => {
    setSelectedPaths([]);
    setPassword('');
    setConfirmPassword('');
    setProgress(null);
    setResult(null);
    setSelectionMode('files');
  };

  const handleSelectFiles = async () => {
    if (!api) return;
    const paths = await api.SelectFiles();
    if (paths && paths.length > 0) {
      setSelectedPaths(paths);
      setSelectionMode('files');
      setResult(null);
    }
  };

  const handleSelectFolder = async () => {
    if (!api) return;
    const path = await api.SelectFolder();
    if (path) {
      setSelectedPaths([path]);
      setSelectionMode('folder');
      setResult(null);
    }
  };

  const handleProcess = async () => {
    if (!api || selectedPaths.length === 0) return;
    if (password.length < 8) {
      alert("Password must be at least 8 characters.");
      return;
    }
    if (activeTab === 'Encrypt' && password !== confirmPassword) {
      alert("Passwords do not match!");
      return;
    }

    setIsProcessing(true);
    setProgress(null);
    setResult(null);

    try {
      let res: OperationResult;

      if (activeTab === 'Encrypt') {
        res = selectionMode === 'folder'
          ? await api.EncryptFolder(selectedPaths[0], password)
          : await api.EncryptFiles(selectedPaths, password);
      } else {
        res = selectionMode === 'folder'
          ? await api.DecryptFolder(selectedPaths[0], password)
          : await api.DecryptFiles(selectedPaths, password);
      }
      setResult(res);
      setSelectedPaths([]);
      setPassword('');
      setConfirmPassword('');
    } catch (err) {
      console.error(err);
      alert("An error occurred during processing.");
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <div className="h-full flex flex-col bg-gray-50 dark:bg-dark-bg text-gray-900 dark:text-gray-100 font-sans transition-colors duration-300">
      
      {/* Top Title Bar / Wails Drag Region */}
      <div className="h-12 wails-drag bg-white/50 dark:bg-dark-surface/50 backdrop-blur-md border-b border-gray-200 dark:border-dark-border flex items-center justify-center shadow-sm z-50 relative">
        <h1 className="font-bold tracking-widest text-transparent bg-clip-text bg-gradient-to-r from-brand-600 to-purple-500 dark:from-brand-400 dark:to-fuchsia-400 text-lg shadow-sm">Reg-X</h1>
      </div>

      <div className="flex flex-1 overflow-hidden">
        {/* Sidebar */}
        <div className="w-48 xl:w-56 bg-white dark:bg-dark-surface border-r border-gray-200 dark:border-dark-border p-4 flex flex-col gap-2">
          {(['Encrypt', 'Decrypt', 'Settings'] as Tab[]).map((tab) => (
            <button
              key={tab}
              onClick={() => {
                if (isProcessing) return;
                setActiveTab(tab);
                clearState();
              }}
              disabled={isProcessing}
              className={`wails-no-drag py-3 px-4 rounded-lg text-left font-medium transition-all ${
                activeTab === tab 
                  ? 'bg-brand-100 dark:bg-brand-900/40 text-brand-700 dark:text-brand-300 shadow-sm' 
                  : 'hover:bg-gray-100 dark:hover:bg-gray-800 text-gray-600 dark:text-gray-400'
              }`}
            >
              {tab}
            </button>
          ))}
          <div className="mt-auto space-y-2 pb-2">
            <button 
              onClick={() => {
                const openURL = (window as any).runtime?.BrowserOpenURL || (window as any).go?.main?.App?.BrowserOpenURL;
                if (openURL) {
                  openURL(atob("aHR0cHM6Ly9naXRodWIuY29tL0p1c3RIYWNrZWRPbg=="));
                } else {
                  window.open(atob("aHR0cHM6Ly9naXRodWIuY29tL0p1c3RIYWNrZWRPbg=="), "_blank");
                }
              }}
              className="w-full text-xs text-center text-brand-500 hover:text-brand-600 dark:text-brand-400 dark:hover:text-brand-300 transition-colors cursor-pointer wails-no-drag"
            >
              by JustHackedOn
            </button>
            <div className="text-xs text-center text-gray-400 dark:text-gray-600">
              Reg-X v1.0.0
            </div>
          </div>
        </div>

        {/* Main Content Area */}
        <div className="flex-1 p-8 overflow-y-auto wails-no-drag relative">
          <div className="max-w-2xl mx-auto glass-panel rounded-2xl p-8 relative">
            
            {/* Header */}
            <div className="mb-8">
              <h2 className="text-2xl font-bold text-gray-800 dark:text-white capitalize">
                {activeTab} files
              </h2>
              <p className="text-gray-500 dark:text-gray-400 mt-1">
                {activeTab === 'Encrypt' 
                  ? 'Secure your sensitive data using AES-256-GCM + Argon2id.' 
                  : activeTab === 'Decrypt' 
                  ? 'Restore your previously encrypted files.'
                  : 'Configure application preferences.'}
              </p>
            </div>

            {/* Content for Encrypt/Decrypt */}
            {(activeTab === 'Encrypt' || activeTab === 'Decrypt') && (
              <div className="space-y-6">
                
                {/* File Selection */}
                {!isProcessing && selectedPaths.length === 0 && !result && (
                  <div className="space-y-4">
                    <div className="drop-zone hover:scale-[1.01] transition-transform">
                      <svg className="w-12 h-12 text-gray-400 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12" />
                      </svg>
                      <p className="text-gray-600 dark:text-gray-300 font-medium">Select items to {activeTab.toLowerCase()}</p>
                      
                      <div className="flex gap-4 mt-6">
                        <button onClick={handleSelectFiles} className="btn-secondary">Choose Files</button>
                        <button onClick={handleSelectFolder} className="btn-secondary">Choose Folder</button>
                      </div>
                    </div>
                  </div>
                )}

                {/* Selected Files State */}
                {!isProcessing && selectedPaths.length > 0 && (
                  <div className="bg-gray-50 dark:bg-dark-bg border border-gray-200 dark:border-dark-border rounded-lg p-4">
                    <div className="flex justify-between items-start">
                      <div>
                        <h4 className="font-medium text-brand-600 dark:text-brand-400">Ready to Process</h4>
                        <p className="text-sm text-gray-500 max-w-sm truncate mt-1">
                          {selectedPaths.length === 1 ? selectedPaths[0] : `${selectedPaths.length} files selected`}
                        </p>
                      </div>
                      <button onClick={() => setSelectedPaths([])} className="text-gray-400 hover:text-red-500 transition-colors">
                         Clear
                      </button>
                    </div>

                    <div className="mt-6 space-y-4">
                      <div>
                        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Password</label>
                        <div className="relative">
                          <input 
                            type={showPassword ? "text" : "password"} 
                            className="styled-input pr-10" 
                            placeholder="Enter highly secure password"
                            value={password}
                            onChange={(e) => setPassword(e.target.value)}
                          />
                          <button 
                            type="button" 
                            onClick={() => setShowPassword(!showPassword)}
                            className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 mt-0.5"
                          >
                            {showPassword ? "Hide" : "Show"}
                          </button>
                        </div>
                      </div>

                      {activeTab === 'Encrypt' && (
                        <div>
                          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">Confirm Password</label>
                          <input 
                            type={showPassword ? "text" : "password"} 
                            className="styled-input" 
                            placeholder="Confirm password"
                            value={confirmPassword}
                            onChange={(e) => setConfirmPassword(e.target.value)}
                          />
                        </div>
                      )}

                      <button 
                        onClick={handleProcess}
                        disabled={password.length < 8}
                        className="btn-primary w-full mt-4 py-3 flex justify-center items-center gap-2"
                      >
                        <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 15v2m-6 4h12a2 2 0 002-2v-6a2 2 0 00-2-2H6a2 2 0 00-2 2v6a2 2 0 002 2zm10-10V7a4 4 0 00-8 0v4h8z" />
                        </svg>
                        Start {activeTab}
                      </button>
                    </div>
                  </div>
                )}

                {/* Processing State */}
                {isProcessing && (
                  <div className="text-center py-10">
                    <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-brand-500 mb-6"></div>
                    <h3 className="text-xl font-medium mb-2">Processing Data</h3>
                    
                    {progress && (
                      <div className="max-w-md mx-auto mt-6 text-left">
                        <div className="flex justify-between text-sm mb-1">
                          <span className="truncate w-48 text-gray-500">{progress.filename}</span>
                          <span className="text-brand-500 font-medium">{Math.round(progress.percent)}%</span>
                        </div>
                        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2.5 mb-2 overflow-hidden">
                          <div className="bg-brand-600 h-2.5 rounded-full transition-all duration-300 ease-out" style={{ width: `${progress.percent}%` }}></div>
                        </div>
                        <div className="flex justify-between text-xs text-gray-400">
                          <span>{progress.current} / {progress.total} files</span>
                          <span>{progress.speed} • ETA: {progress.eta}</span>
                        </div>
                      </div>
                    )}
                  </div>
                )}

                {/* Result State */}
                {!isProcessing && result && (
                  <div className={`p-6 rounded-xl border ${result.failed > 0 ? 'border-red-200 bg-red-50 dark:bg-red-900/10 dark:border-red-900/50' : 'border-green-200 bg-green-50 dark:bg-green-900/10 dark:border-green-900/50'}`}>
                    <h3 className={`text-lg font-bold flex items-center gap-2 ${result.failed > 0 ? 'text-red-700 dark:text-red-400' : 'text-green-700 dark:text-green-400'}`}>
                      {result.failed > 0 ? (
                        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                      ) : (
                        <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
                      )}
                      Operation Complete
                    </h3>
                    <div className="mt-4 grid grid-cols-2 gap-4 text-sm">
                      <div className="bg-white dark:bg-dark-bg p-3 rounded shadow-sm">
                        <p className="text-gray-500">Successfully Processed</p>
                        <p className="text-2xl font-semibold text-green-600">{result.success}</p>
                      </div>
                      <div className="bg-white dark:bg-dark-bg p-3 rounded shadow-sm">
                        <p className="text-gray-500">Failed Items</p>
                        <p className={`text-2xl font-semibold ${result.failed > 0 ? 'text-red-600' : 'text-gray-700 dark:text-gray-300'}`}>{result.failed}</p>
                      </div>
                    </div>
                    <p className="text-xs text-gray-500 mt-4 text-right">Elapsed time: {result.elapsedTime}</p>
                    
                    {result.errors && result.errors.length > 0 && (
                      <div className="mt-4 max-h-32 overflow-y-auto bg-white/50 dark:bg-dark-bg rounded p-2 border border-red-100 dark:border-red-900/30">
                        {result.errors.map((err, i) => (
                           <div key={i} className="text-xs text-red-500 mb-1 font-mono">{err}</div>
                        ))}
                      </div>
                    )}
                    
                    <button onClick={clearState} className="btn-secondary w-full mt-6 py-2">
                      Process More Files
                    </button>
                  </div>
                )}
                
              </div>
            )}

            {/* Content for Settings */}
            {activeTab === 'Settings' && (
              <div className="space-y-6">
                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Encrypted File Extension</label>
                  <p className="text-xs text-gray-500 mb-2">The extension appended to files you encrypt. Default is .pse</p>
                  <input 
                    type="text" 
                    className="styled-input max-w-xs" 
                    value={settings.extension} 
                    onChange={e => handleSaveSettings({...settings, extension: e.target.value})}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Appearance UI Theme</label>
                  <select 
                    className="styled-select max-w-xs"
                    value={settings.theme}
                    onChange={e => handleSaveSettings({...settings, theme: e.target.value})}
                  >
                    <option value="dark">Dark Mode</option>
                    <option value="light">Light Mode</option>
                  </select>
                </div>
                
                <div className="pt-4 border-t border-gray-200 dark:border-dark-border">
                  <label className="flex items-center space-x-3 cursor-pointer">
                    <input 
                      type="checkbox" 
                      className="w-5 h-5 text-brand-600 rounded focus:ring-brand-500"
                      checked={settings.deleteOriginals}
                      onChange={e => handleSaveSettings({...settings, deleteOriginals: e.target.checked})}
                    />
                    <div>
                      <span className="block text-sm font-medium text-gray-700 dark:text-gray-300">Delete originals after encryption</span>
                      <span className="block text-xs text-gray-500">Automatically delete the source file if encryption completes successfully.</span>
                    </div>
                  </label>
                </div>

                <div className="pt-4 border-t border-gray-200 dark:border-dark-border">
                  <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Default Output Folder</label>
                  <p className="text-xs text-gray-500 mb-2">Leave blank to save files right next to the originals.</p>
                  <div className="flex gap-2 max-w-md">
                    <input 
                      type="text" 
                      readOnly
                      className="styled-input flex-1 bg-gray-50 text-gray-500" 
                      value={settings.outputFolder || "(Alongside Source)"} 
                    />
                    <button 
                      className="btn-secondary whitespace-nowrap mt-1"
                      onClick={async () => {
                        if (api) {
                          const folder = await api.SelectOutputFolder();
                          if (folder) handleSaveSettings({...settings, outputFolder: folder});
                        }
                      }}
                    >
                      Browse
                    </button>
                    {settings.outputFolder && (
                      <button 
                        className="text-red-500 hover:text-red-600 px-2 mt-1"
                        onClick={() => handleSaveSettings({...settings, outputFolder: ''})}
                      >
                        Reset
                      </button>
                    )}
                  </div>
                </div>

              </div>
            )}
            
          </div>
        </div>
      </div>
    </div>
  );
}
