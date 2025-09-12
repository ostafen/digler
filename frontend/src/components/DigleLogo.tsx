interface DigleLogoProps {
  size?: "sm" | "lg";
  showTagline?: boolean;
}

export const DigleLogo = ({ size = "lg", showTagline = true }: DigleLogoProps) => {
  return (
    <div className="flex flex-col items-center">
      <div className="flex items-center gap-3">
        <div className={`${size === "lg" ? "h-16 w-16" : "h-12 w-12"}`}>
          <img 
            src="/lovable-uploads/f64971ef-af26-4710-aba1-43092c2d604f.png"
            alt="Digler Logo - Mole with magnifying glass"
            className="w-full h-full object-contain"
          />
        </div>
        {showTagline && (
          <div className="flex flex-col">
            <p className={`text-muted-foreground font-medium ${size === "lg" ? "text-sm" : "text-xs"}`}>
              Go Deep. Get Back Your Data.
            </p>
          </div>
        )}
      </div>
    </div>
  );
};