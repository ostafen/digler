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
  const [currentScreen, setCurrentScreen] = useState<Screen>("home");
  const [scanResults, setScanResults] = useState<{ filesFound: number; path: string, scanId: string } | null>(null);
  const [selectedFiles, setSelectedFiles] = useState<{ scanId: string, selectedFiles: FileItem[] } | null>(null);

  const navigateToScan = () => setCurrentScreen("scan");
  const navigateToResults = (results: { filesFound: number; path: string, scanId: string }) => {
    setScanResults(results);
    setCurrentScreen("results");
  };

  const navigateToRecovery = (results: { scanId: string, selectedFiles: FileItem[] }) => {
    setSelectedFiles(results);
    setCurrentScreen("recovery");
  };

  const navigateHome = () => {
    setCurrentScreen("home");
    setScanResults(null);
    setSelectedFiles(null);
  };

  const handleOpenRecent = (scanId: string) => {
    // Mock opening recent scan
    setScanResults({ filesFound: 0, path: "/Users/john/disk_image.dd", scanId: "" });
    setCurrentScreen("results");
  };

  const renderCurrentScreen = () => {
    switch (currentScreen) {
      case "scan":
        return (
          <Scan
            onBack={navigateHome}
            onScanComplete={navigateToResults}
          />
        );
      case "results":
        return (
          <Results
            onBack={() => setCurrentScreen("scan")}
            onStartRecovery={navigateToRecovery}
            scanResults={scanResults!}
          />
        );
      case "recovery":
        return (
          <Recovery
            scanID={selectedFiles.scanId}
            onBack={() => setCurrentScreen("results")}
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
