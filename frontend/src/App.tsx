import { Navigate, Route, Routes } from "react-router-dom";
import Shell from "./components/Shell";
import Overview from "./pages/Overview";
import ImportPage from "./pages/Import";
import Endpoints from "./pages/Endpoints";
import Requests from "./pages/Requests";

export default function App() {
  return (
    <Routes>
      <Route element={<Shell />}>
        <Route path="/" element={<Overview />} />
        <Route path="/import" element={<ImportPage />} />
        <Route path="/endpoints" element={<Endpoints />} />
        <Route path="/requests" element={<Requests />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Route>
    </Routes>
  );
}
