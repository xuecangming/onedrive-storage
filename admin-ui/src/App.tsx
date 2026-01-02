import React from 'react';
import { BrowserRouter as Router, Routes, Route, Navigate } from 'react-router-dom';
import MainLayout from './components/MainLayout';
import Dashboard from './pages/Dashboard';
import Buckets from './pages/Buckets';
import Accounts from './pages/Accounts';
import Audit from './pages/Audit';

const App: React.FC = () => {
  return (
    <Router>
      <MainLayout>
        <Routes>
          <Route path="/" element={<Navigate to="/dashboard" replace />} />
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/buckets" element={<Buckets />} />
          <Route path="/accounts" element={<Accounts />} />
          <Route path="/audit" element={<Audit />} />
        </Routes>
      </MainLayout>
    </Router>
  );
};

export default App;
