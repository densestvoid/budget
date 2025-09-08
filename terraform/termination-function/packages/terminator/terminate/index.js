exports.main = async (args) => {
  const { DO_TOKEN, TARGET_APP_ID, TERMINATION_TIME } = args;
  
  console.log('=== Budget App Auto-Termination Function ===');
  console.log(`Function triggered at: ${new Date().toISOString()}`);
  console.log(`Target App ID: ${TARGET_APP_ID}`);
  console.log(`Scheduled termination time: ${TERMINATION_TIME}`);
  
  if (!DO_TOKEN || !TARGET_APP_ID) {
    const error = 'Missing required parameters: DO_TOKEN and TARGET_APP_ID';
    console.error(error);
    return { error };
  }

  try {
    const now = new Date();
    const terminationTime = new Date(TERMINATION_TIME);
    
    console.log(`Current time: ${now.toISOString()}`);
    console.log(`Termination time: ${terminationTime.toISOString()}`);
    
    // Check if termination time has arrived (with 2-minute tolerance for cron precision)
    const timeDiff = (now - terminationTime) / (1000 * 60); // minutes
    console.log(`Time difference: ${timeDiff.toFixed(2)} minutes`);
    
    if (timeDiff >= -1 && timeDiff <= 5) { // Between 1 minute early and 5 minutes late
      console.log('✅ Termination time reached, proceeding with app deletion...');
      
      // First, verify the app still exists
      const checkResponse = await fetch(`https://api.digitalocean.com/v2/apps/${TARGET_APP_ID}`, {
        headers: { 
          'Authorization': `Bearer ${DO_TOKEN}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (!checkResponse.ok) {
        if (checkResponse.status === 404) {
          console.log('ℹ️  App already deleted or not found');
          return { 
            success: true, 
            action: 'already_deleted',
            app_id: TARGET_APP_ID,
            message: 'App was already deleted or not found'
          };
        } else {
          throw new Error(`Failed to check app status: ${checkResponse.status}`);
        }
      }
      
      const appData = await checkResponse.json();
      console.log(`App found: ${appData.app.spec.name}, Status: ${appData.app.phase}`);
      
      // Delete the app
      console.log(`Deleting app ${TARGET_APP_ID}...`);
      const deleteResponse = await fetch(`https://api.digitalocean.com/v2/apps/${TARGET_APP_ID}`, {
        method: 'DELETE',
        headers: { 
          'Authorization': `Bearer ${DO_TOKEN}`,
          'Content-Type': 'application/json'
        }
      });
      
      if (deleteResponse.ok) {
        console.log(`✅ Successfully terminated app ${TARGET_APP_ID}`);
        
        // Get the current function's app ID for self-destruction
        const functionAppId = args.__ow_meta?.app_id;
        
        if (functionAppId && functionAppId !== TARGET_APP_ID) {
          console.log(`Self-destructing termination function ${functionAppId}...`);
          
          // Small delay to ensure this response is sent before self-destruction
          setTimeout(async () => {
            await fetch(`https://api.digitalocean.com/v2/apps/${functionAppId}`, {
              method: 'DELETE',
              headers: { 
                'Authorization': `Bearer ${DO_TOKEN}`,
                'Content-Type': 'application/json'
              }
            });
            console.log(`🗑️ Termination function ${functionAppId} self-destructed`);
          }, 2000);
        }
        
        return { 
          success: true, 
          action: 'terminated',
          app_id: TARGET_APP_ID,
          terminated_at: now.toISOString(),
          function_self_destructed: !!functionAppId
        };
      } else {
        const errorText = await deleteResponse.text();
        throw new Error(`Failed to delete app: ${deleteResponse.status} - ${errorText}`);
      }
    } else if (timeDiff < -1) {
      const minutesRemaining = Math.abs(timeDiff);
      console.log(`⏰ Termination not due yet. ${minutesRemaining.toFixed(1)} minutes remaining.`);
      
      return {
        success: true,
        action: 'waiting',
        app_id: TARGET_APP_ID,
        minutes_remaining: minutesRemaining,
        termination_scheduled_for: terminationTime.toISOString()
      };
    } else {
      console.log(`⚠️  Termination window missed (${timeDiff.toFixed(1)} minutes late). App may have been manually deleted.`);
      
      return {
        success: true,
        action: 'missed_window',
        app_id: TARGET_APP_ID,
        minutes_late: timeDiff
      };
    }
    
  } catch (error) {
    console.error('Function execution error:', error);
    return { 
      error: error.message,
      app_id: TARGET_APP_ID,
      timestamp: new Date().toISOString()
    };
  }
};