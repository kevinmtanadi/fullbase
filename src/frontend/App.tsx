import Sidebar from "./Sidebar";
import { Navigate, Outlet } from "react-router-dom";
import { ToastContainer } from "react-toastify";
import "react-toastify/dist/ReactToastify.css";
import { TbCode } from "react-icons/tb";
import useIsAuthenticated from "react-auth-kit/hooks/useIsAuthenticated";
import { CgDatabase } from "react-icons/cg";
import { LuFunctionSquare, LuUsers2 } from "react-icons/lu";
import { useQuery } from "@tanstack/react-query";
import axiosInstance from "./pkg/axiosInstance";
import { BiWrench } from "react-icons/bi";
import { AiOutlineTable } from "react-icons/ai";

const tabs = [
  {
    name: "Database",
    path: "/",
    icon: AiOutlineTable,
  },
  {
    name: "SQL Editor",
    path: "/sql",
    icon: TbCode,
  },
  {
    name: "Functions",
    path: "/function",
    icon: LuFunctionSquare,
  },
  {
    name: "Storage",
    path: "/storage",
    icon: CgDatabase,
  },
  {
    name: "Backup",
    path: "/backup",
    icon: CgDatabase,
  },
  {
    name: "Admins",
    path: "/admin",
    icon: LuUsers2,
  },
  {
    name: "Settings",
    path: "/setting",
    icon: BiWrench,
  },
];

const checkAuth = () => {
  var isMounted = false;
  const { data: admin } = useQuery<any>({
    queryKey: ["admin"],
    queryFn: async () => {
      const { data } = await axiosInstance.get("/api/admin").then((res) => {
        isMounted = true;
        return res.data;
      });

      return data || [];
    },
  });

  if (!isMounted) {
    return <>Loading...</>;
  }

  if (admin.rows.length === 0) {
    console.log(admin.rows.length);
    return <Navigate to="/signup" />;
  }

  const isAuth = useIsAuthenticated();
  if (admin?.rows.length > 0) {
    if (!isAuth) {
      const maxAttempts = 10;
      let attempts = 0;

      const checkAuthentication = () => {
        attempts++;
        if (!isAuth && attempts < maxAttempts) {
          setTimeout(checkAuthentication, 100);
        } else {
          if (!isAuth) {
            return <Navigate to="/login" />;
          }
        }
      };

      checkAuthentication();
    }
  }
};

function App() {
  checkAuth();

  return (
    <>
      <div className="max-h-screen overflow-hidden h-screen w-full flex">
        <div className="">
          <Sidebar tabs={tabs} />
        </div>
        <div className="w-full">
          <Outlet />
          <ToastContainer draggable position="bottom-center" />
        </div>
      </div>
    </>
  );
}

export default App;
