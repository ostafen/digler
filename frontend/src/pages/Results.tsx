import { useEffect, useState } from "react";
import { motion } from "framer-motion";
import {
  ArrowLeft,
  Folder,
  File,
  Image,
  FileText,
  Archive,
  Music,
  Video,
  Search,
  Filter,
  Download,
  Eye,
  CheckSquare,
  Square
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { DigleLogo } from "@/components/DigleLogo";
import { api } from '../wailsjs/go/models';
import { ScanResult, FileContent } from '../wailsjs/go/api/ScanAPI';
import { formatFileSize } from '../lib/utils';

const enum FileType {
  Image = "image",
  Document = "document",
  Archive = "archive",
  Audio = "audio",
  Video = "video",
  Other = "other"
}

interface FileItem {
  id: string;
  name: string;
  path: string;
  size: string;
  type: FileType;
  status: "recoverable" | "corrupted" | "partial";
  preview?: string;
}

interface FolderItem {
  id: string;
  name: string;
  path: string;
  children: (FolderItem | FileItem)[];
  isExpanded?: boolean;
}

interface ResultsProps {
  onBack: () => void;
  onStartRecovery: (results: { scanId: string, selectedFiles: FileItem[] }) => void;
  scanResults: { scanId: string, filesFound: number, path: string };
}

const fileType = (ext: string): FileType => {
  switch (ext.toLowerCase()) {
    case "jpg":
    case "jpeg":
    case "png":
    case "gif":
    case "bmp":
    case "tiff":
    case "svg":
      return FileType.Image;

    case "pdf":
    case "doc":
    case "docx":
    case "xls":
    case "xlsx":
    case "ppt":
    case "pptx":
    case "txt":
    case "md":
      return FileType.Document;

    case "zip":
    case "rar":
    case "7z":
    case "tar":
    case "gz":
      return FileType.Archive;

    case "mp3":
    case "wav":
    case "flac":
    case "aac":
      return FileType.Audio;

    case "mp4":
    case "mkv":
    case "avi":
    case "mov":
      return FileType.Video;

    default:
      return FileType.Other;
  }
}

// Utility to get MIME type from file extension
const getMimeType = (fileName: string) => {
  const ext = fileName.split('.').pop()?.toLowerCase();
  switch (ext) {
    // Images
    case 'png': return 'image/png';
    case 'jpg':
    case 'jpeg': return 'image/jpeg';
    case 'gif': return 'image/gif';
    case 'bmp': return 'image/bmp';
    case 'tiff': return 'image/tiff';
    case 'svg': return 'image/svg+xml';

    // Audio
    case 'mp3': return 'audio/mpeg';
    case 'wav': return 'audio/wav';
    case 'flac': return 'audio/flac';
    case 'aac': return 'audio/aac';
    case 'ogg': return 'audio/ogg';

    default: return ''; // unsupported type
  }
};

// Fetch and build data URL
const getImageDataUrl = async (scanId: string, fileName: string) => {
  const base64Content = await FileContent(scanId, fileName); // your API call
  const mimeType = getMimeType(fileName);

  if (!mimeType) return null; // not an image

  return `data:${mimeType};base64,${base64Content}`;
};

const getFileIcon = (type: string, status: string) => {
  const baseClasses = "h-4 w-4";
  const statusColor = status === "recoverable" ? "text-success" :
    status === "partial" ? "text-warning" : "text-danger";

  switch (type) {
    case "image": return <Image className={`${baseClasses} ${statusColor}`} />;
    case "document": return <FileText className={`${baseClasses} ${statusColor}`} />;
    case "archive": return <Archive className={`${baseClasses} ${statusColor}`} />;
    case "audio": return <Music className={`${baseClasses} ${statusColor}`} />;
    case "video": return <Video className={`${baseClasses} ${statusColor}`} />;
    default: return <File className={`${baseClasses} ${statusColor}`} />;
  }
};

const renderPreviewContent = (file: FileItem) => {
  const defaultPreview = (
    <div className="aspect-square bg-muted rounded-lg flex items-center justify-center">
      <div className="text-center text-muted-foreground">
        {getFileIcon(file.type, file.status)}
        <p className="mt-2 text-sm">No preview available</p>
      </div>
    </div>
  );

  if (!file.preview)
    return defaultPreview;

  switch (file.type) {
    case FileType.Image:
      return (
        <div className="aspect-square bg-muted rounded-lg flex items-center justify-center">
          <img
            src={file.preview}
            alt={file.name}
            className="max-w-full max-h-full object-contain"
          />
        </div>
      );

    case FileType.Audio:
      return (
        <audio controls className="w-full">
          <source src={file.preview} type="audio/mpeg" />
          Your browser does not support the audio element.
        </audio>
      );

    default:
      return defaultPreview;
  }
};

export const Results = ({ onBack, onStartRecovery, scanResults }: ResultsProps) => {
  const [selectedFiles, setSelectedFiles] = useState<Set<string>>(new Set());
  const [searchTerm, setSearchTerm] = useState("");
  const [filterType, setFilterType] = useState("all");
  const [previewFile, setPreviewFile] = useState<FileItem | null>(null);
  const [fileTree, setFileTree] = useState<FolderItem[]>([]);
  const [isPreviewLoading, setIsPreviewLoading] = useState(false);

  useEffect(() => {
    // disable/enable scrolling on the whole page
    document.body.style.overflowY = 'hidden';

    return () => {
      document.body.style.overflowY = 'unset';
    };
  }, []);

  const handlePreview = async (file: FileItem) => {
    setIsPreviewLoading(true); // start loading
    const previewContent = await getImageDataUrl(scanResults.scanId, file.name);
    setPreviewFile({
      ...file,
      preview: previewContent,
    });
    setIsPreviewLoading(false); // finished loading
  };

  useEffect(() => {
    const fetchScanResults = async () => {
      let res: api.ScanResultResponse = null
      try {
        res = await ScanResult(scanResults.scanId)
      } catch (error) {
        console.error("Failed to fetch scan results:", error);
        return;
      }

      const allFiles = res.files.map(file => {
        return {
          id: file.name,
          name: file.name,
          path: file.name,
          size: formatFileSize(file.size),
          type: fileType(file.ext),
          status: "recoverable" as "recoverable",
        }
      });

      const images = allFiles.filter(file => file.type == FileType.Image)
      const documents = allFiles.filter(file => file.type == FileType.Document)
      const audio = allFiles.filter(file => file.type == FileType.Audio)
      const other = allFiles.filter(file => file.type == FileType.Other)

      const children = [
        {
          id: "documents",
          name: "Documents",
          path: "/Documents",
          children: documents
        },
        {
          id: "images",
          name: "Images",
          path: "/Images",
          children: images
        },
        {
          id: "audio",
          name: "Audio",
          path: "/Audio",
          children: audio
        },
        {
          id: "Other",
          name: "Other",
          path: "/Other",
          children: other
        },
      ]

      setFileTree([
        {
          id: "root",
          name: "Recovered Files",
          path: "/",
          children: children.filter(item => item.children.length > 0)
        }
      ]);

      // Selected all files by default
      setSelectedFiles(new Set(allFiles.map(f => f.id)))
    }
    fetchScanResults();
  }, []);


  const getStatusBadge = (status: string) => {
    switch (status) {
      case "recoverable":
        return <Badge className="bg-success/10 text-success border-success/20">Recoverable</Badge>;
      case "partial":
        return <Badge className="bg-warning/10 text-warning border-warning/20">Partial</Badge>;
      case "corrupted":
        return <Badge className="bg-danger/10 text-danger border-danger/20">Corrupted</Badge>;
      default:
        return null;
    }
  };

  const toggleFileSelection = (fileId: string) => {
    const newSelection = new Set(selectedFiles);
    if (newSelection.has(fileId)) {
      newSelection.delete(fileId);
    } else {
      newSelection.add(fileId);
    }
    setSelectedFiles(newSelection);
  };

  const renderTreeItem = (item: FolderItem | FileItem, level = 0) => {
    const isFile = 'size' in item;
    const paddingLeft = `${level * 1.5}rem`;

    if (isFile) {
      const file = item as FileItem;
      const isSelected = selectedFiles.has(file.id);

      return (
        <div
          key={file.id}
          className="flex items-center gap-2 p-2 hover:bg-muted/30 cursor-pointer rounded"
          style={{ paddingLeft }}
          onClick={() => toggleFileSelection(file.id)}
        >
          {isSelected ?
            <CheckSquare className="h-4 w-4 text-primary" /> :
            <Square className="h-4 w-4 text-muted-foreground" />
          }
          {getFileIcon(file.type, file.status)}
          <span className="flex-1 text-sm">{file.name}</span>
          <span className="text-xs text-muted-foreground">{file.size}</span>
          {getStatusBadge(file.status)}
          <Button
            variant="ghost"
            size="sm"
            className="p-1"
            onClick={async (e) => {
              e.stopPropagation();
              handlePreview(file);
            }}
          >
            <Eye className="h-3 w-3" />
          </Button>
        </div>
      );
    }

    // Collect all file IDs in a subtree
    const getAllFileIds = (item: FolderItem | FileItem): string[] => {
      if ('size' in item) {
        return [item.id]; // it's a file
      }
      return item.children.flatMap(child => getAllFileIds(child));
    };

    // Determine selection state of a folder
    const getFolderSelectionState = (folder: FolderItem): "all" | "none" | "partial" => {
      const allFileIds = getAllFileIds(folder);
      const selectedCount = allFileIds.filter(id => selectedFiles.has(id)).length;

      if (selectedCount === 0) return "none";
      if (selectedCount === allFileIds.length) return "all";
      return "partial";
    };

    // Modified toggle
    const toggleItemSelection = (item: FolderItem | FileItem) => {
      const newSelection = new Set(selectedFiles);

      if ('size' in item) {
        // File
        if (newSelection.has(item.id)) {
          newSelection.delete(item.id);
        } else {
          newSelection.add(item.id);
        }
      } else {
        // Folder
        const allFileIds = getAllFileIds(item);
        const state = getFolderSelectionState(item);

        if (state === "all") {
          // deselect everything
          allFileIds.forEach(id => newSelection.delete(id));
        } else {
          // select everything
          allFileIds.forEach(id => newSelection.add(id));
        }
      }
      setSelectedFiles(newSelection);
    };

    const folder = item as FolderItem;
    const folderState = getFolderSelectionState(folder);

    return (
      <div key={folder.id}>
        <div
          className="flex items-center gap-2 p-2 hover:bg-muted/30 cursor-pointer rounded font-medium"
          style={{ paddingLeft }}
          onClick={() => toggleItemSelection(folder)}
        >
          {folderState === "all" && <CheckSquare className="h-4 w-4 text-primary" />}
          {folderState === "none" && <Square className="h-4 w-4 text-muted-foreground" />}
          {folderState === "partial" && (
            <Square className="h-4 w-4 text-warning" /> // or render a custom "indeterminate" checkbox
          )}
          <Folder className="h-4 w-4 text-info" />
          <span className="text-sm">{folder.name}</span>
          <span className="text-xs text-muted-foreground ml-auto">
            {getAllFileIds(folder).length} files
          </span>
        </div>
        {folder.children.map(child => renderTreeItem(child, level + 1))}
      </div>
    );
  };

  return (
    <div className="min-h-screen bg-gradient-subtle">
      <div className="container mx-auto px-6 py-8 max-w-7xl">
        {/* Header */}
        <motion.div
          className="flex items-center justify-between mb-8"
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
        >
          <div className="flex items-center gap-4">
            <Button variant="ghost" onClick={onBack} className="p-2">
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <DigleLogo size="sm" showTagline={false} />
            <div>
              <h2 className="text-xl font-semibold">Scan Results</h2>
              <p className="text-sm text-muted-foreground">
                Found {scanResults.filesFound.toLocaleString()} recoverable files
              </p>
            </div>
          </div>
          <Button
            className="hero-gradient text-primary-foreground"
            onClick={() => {
              const files = Array.from(selectedFiles).map(id =>
                fileTree[0].children.flatMap(folder =>
                  'children' in folder ? folder.children : [folder]
                ).find(item => item.id === id) as FileItem
              ).filter(Boolean);
              onStartRecovery({ scanId: scanResults.scanId, selectedFiles: files });
            }}
            disabled={selectedFiles.size === 0}
          >
            <Download className="mr-2 h-4 w-4" />
            Recover Selected ({selectedFiles.size})
          </Button>
        </motion.div>

        <div className="grid grid-cols-12 gap-6 h-[calc(100vh-200px)] overflow-hidden pb-5">
          {/* File Browser */}
          <motion.div
            className="col-span-8"
            style={{ maxHeight: '700px' }}
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
          >
            <Card className="h-full">
              <CardHeader className="pb-3">
                <div className="flex items-center justify-between">
                  <CardTitle>Recovered Files</CardTitle>
                  <div className="flex items-center gap-2">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder="Search files..."
                        className="pl-10 w-64"
                        value={searchTerm}
                        onChange={(e) => setSearchTerm(e.target.value)}
                      />
                    </div>
                    <Select value={filterType} onValueChange={setFilterType}>
                      <SelectTrigger className="w-32">
                        <Filter className="h-4 w-4 mr-2" />
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="all">All Files</SelectItem>
                        <SelectItem value="image">Images</SelectItem>
                        <SelectItem value="document">Documents</SelectItem>
                        <SelectItem value="archive">Archives</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </CardHeader>
              <CardContent className="p-0 h-[calc(100%-80px)]" >
                <div className="overflow-y-auto h-full p-4">
                  {fileTree.map(item => renderTreeItem(item))}
                </div>
              </CardContent>
            </Card>
          </motion.div>

          {/* Preview Panel */}
          <motion.div
            className="col-span-4"
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <Card className="h-full">
              <CardHeader>
                <CardTitle>File Preview</CardTitle>
              </CardHeader>
              <CardContent>
                {isPreviewLoading ? (
                  <div className="h-full flex items-center justify-center">
                    <div className="animate-spin rounded-full h-10 w-10 border-t-2 border-b-2 border-primary"></div>
                  </div>
                ) : previewFile ? (
                  <div className="space-y-4">
                    {renderPreviewContent(previewFile)}
                    <div className="space-y-2">
                      <h3 className="font-medium truncate">{previewFile.name}</h3>
                      <p className="text-sm text-muted-foreground">{previewFile.path}</p>
                      <div className="flex justify-between text-sm">
                        <span>Size:</span>
                        <span>{previewFile.size}</span>
                      </div>
                      <div className="flex justify-between items-center">
                        <span className="text-sm">Status:</span>
                        {getStatusBadge(previewFile.status)}
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className="h-full flex items-center justify-center text-center text-muted-foreground">
                    <div>
                      <Eye className="h-12 w-12 mx-auto mb-4 opacity-50" />
                      <p>Select a file to preview</p>
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          </motion.div>
        </div>
      </div >
    </div >
  );
};