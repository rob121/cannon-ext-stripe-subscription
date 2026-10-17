<?php
/**
 * @version    CVS: 1.0.0
 * @package    Com_Subscription
 * @author     Rob <pr@floridarangers.com>
 * @copyright  2024 Rob
 * @license    GNU General Public License version 2 or later; see LICENSE.txt
 */

namespace Sub\Component\Subscription\Site\Helper;

defined('_JEXEC') or die;

use \Joomla\CMS\Factory;
use \Joomla\CMS\MVC\Model\BaseDatabaseModel;
use \Joomla\CMS\Component\ComponentHelper;

/**
 * Class SubscriptionFrontendHelper
 *
 * @since  1.0.0
 */
class SubscriptionHelper
{
	

	/**
	 * Gets the files attached to an item
	 *
	 * @param   int     $pk     The item's id
	 *
	 * @param   string  $table  The table's name
	 *
	 * @param   string  $field  The field's name
	 *
	 * @return  array  The files
	 */
	 
	public static function getStripeKeys($product_id){ 
		
		
		$app = Factory::getApplication();
		
		$model = $app->bootComponent('com_subscription')->getMVCFactory()->createModel('Product', 'Site');
		
		$item       = $model->getItem($product_id);		
		
		return self::getStripeKeysMode($item->test_mode);
		

		
	} 
	
	
	public static function getStripeKeysMode($test_mode=0){ 
		
		$obj = new \stdClass;		
		
		$params = ComponentHelper::getParams("com_subscription");
	
		
		if ($test_mode==1){
			
			$obj->server = $params->get("server_stripe_key_test");
			$obj->js = $params->get("js_stripe_key_test");
		
		}else{
			
			$obj->server = $params->get("server_stripe_key_live");
			$obj->js = $params->get("js_stripe_key_live");
		}
		
		return $obj;
		
		
	} 
	
	 
	public static function getStripeCustomer($product_id){ 
		
		$app = Factory::getApplication();
		
		$params = ComponentHelper::getParams("com_subscription");
					
		$model = $app->bootComponent('com_subscription')->getMVCFactory()->createModel('Product', 'Site');
	
		$item       = $model->getItem($product_id);		
		$user       = $app->getIdentity();
		
		
		$obj = new \stdClass;
		
		if($user->get('id') < 1) { 
			
			return false;
			
		}
	
		
		$obj = self::getStripeKeysMode($item->test_mode);
		
		//we need to lookup the subscriptions for this user and see if there are any customer id's
		
		$stripe = new \Stripe\StripeClient($obj->server);
				
		$customer = $stripe->customers->all(['email' => $user->get('email')]);
		 
		
		if (count($customer->data)>0){ 
			
			$cust = $customer->data[0];
			
		}else{
			
		$cust = $stripe->customers->create([
		  'email' => $user->get("email"),
		  'name' => $user->get("name")]);
		  
		}  
		
	    return $cust;
	
		
	} 
	
	
	public static function getFiles($pk, $table, $field)
	{
		$db = Factory::getContainer()->get('DatabaseDriver');
		$query = $db->getQuery(true);

		$query
			->select($field)
			->from($table)
			->where('id = ' . (int) $pk);

		$db->setQuery($query);

		return explode(',', $db->loadResult());
	}

	/**
	 * Gets the edit permission for an user
	 *
	 * @param   mixed  $item  The item
	 *
	 * @return  bool
	 */
	public static function canUserEdit($item)
	{
		$permission = false;
		$user       = Factory::getApplication()->getIdentity();

		if ($user->authorise('core.edit', 'com_subscription') || (isset($item->created_by) && $user->authorise('core.edit.own', 'com_subscription') && $item->created_by == $user->id) || $user->authorise('core.create', 'com_subscription'))
		{
			$permission = true;
		}

		return $permission;
	}
}
