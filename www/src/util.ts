export const statusColor = (status: string) => {
  if (status.startsWith("Error") || status === "Failed") return "danger";
  return "success";
};
