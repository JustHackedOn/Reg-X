// This file declares the Wails auto-generated bindings so TypeScript doesn't throw errors
// before `wails dev` or `wails build` runs for the first time.

declare namespace window {
  namespace go {
    namespace main {
      namespace App {
        function EncryptFiles(paths: string[], password: string): Promise<any>;
        function EncryptFolder(path: string, password: string): Promise<any>;
        function DecryptFiles(paths: string[], password: string): Promise<any>;
        function DecryptFolder(path: string, password: string): Promise<any>;
        function SelectFiles(): Promise<string[]>;
        function SelectFolder(): Promise<string>;
        function SelectOutputFolder(): Promise<string>;
        function GetSettings(): Promise<any>;
        function SaveSettings(settings: any): Promise<void>;
        function GetAppVersion(): Promise<string>;
      }
    }
  }
  function wailsRuntimeEventsOn(eventName: string, callback: (data: any) => void): void;
}
