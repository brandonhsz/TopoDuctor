export type ProjectsFile = {
  paths: string[];
  active: string;
  preferred_branches?: Record<string, string[]>;
};
