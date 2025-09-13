import { useState } from "react";
import { Toaster } from "@/components/ui/toaster";
import { Toaster as Sonner } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Home } from "./pages/Home";
import { Scan } from "./pages/Scan";
import { Results } from "./pages/Results";
import { Recovery } from "./pages/Recovery";

const queryClient = new QueryClient();

type Screen = "home" | "scan" | "results" | "recovery";

interface FileItem {
  id: string;
  name: string;
  path: string;
  size: string;
  type: "image" | "document" | "archive" | "audio" | "video" | "other";
  status: "recoverable" | "corrupted" | "partial";
  preview?: string;
}

const App = () => {
  const [_, setScreenHistory] = useState<Screen[]>([]);
  const [currentScreen, setCurrentScreen] = useState<Screen>("home");
  const [scanMode, setScanMode] = useState<'image' | 'device'>('image');
  const [scanResults, setScanResults] = useState<{ filesFound: number; path: string, scanId: string } | null>(null);
  const [selectedFiles, setSelectedFiles] = useState<{ scanId: string, selectedFiles: FileItem[] } | null>(null);

  const navigateToScreen = (screen: Screen) => {
    setScreenHistory(prev => [...prev, currentScreen]);
    setCurrentScreen(screen);
  }

  const navigateToScan = (mode: 'image' | 'device') => {
    setScanMode(mode);
    navigateToScreen("scan");
  }

  const navigateToResults = (results: { filesFound: number; path: string, scanId: string }) => {
    setScanResults(results);
    navigateToScreen("results");
  };

  const navigateToRecovery = (results: { scanId: string, selectedFiles: FileItem[] }) => {
    setSelectedFiles(results);
    navigateToScreen("recovery");
  };

  const navigateHome = () => {
    navigateToScreen("home");
    setScanResults(null);
    setSelectedFiles(null);
  };

  const handleOpenRecent = (scanId: string) => {
    setScanResults({ filesFound: 0, path: "", scanId: scanId });
    navigateToScreen("results");
  };

  const onBack = () => {
    setScreenHistory(prev => {
      if (prev.length === 0) return prev;
      const lastScreen = prev[prev.length - 1];
      setCurrentScreen(lastScreen);
      return prev.slice(0, -1);
    });
  };

  const renderCurrentScreen = () => {
    switch (currentScreen) {
      case "scan":
        return (
          <Scan
            onBack={onBack}
            mode={scanMode}
            onScanComplete={navigateToResults}
          />
        );
      case "results":
        return (
          <Results
            onBack={onBack}
            onStartRecovery={navigateToRecovery}
            scanResults={scanResults!}
          />
        );
      case "recovery":
        return (
          <Recovery
            scanID={selectedFiles.scanId}
            onBack={onBack}
            onComplete={navigateHome}
            selectedFiles={selectedFiles.selectedFiles}
          />
        );
      default:
        return (
          <Home
            onNavigateToScan={navigateToScan}
            onOpenRecent={handleOpenRecent}
          />
        );
    }
  };

  return (
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <Toaster />
        <Sonner />
        {renderCurrentScreen()}
      </TooltipProvider>
    </QueryClientProvider>
  );
};

export default App;
