<?php
/**
 * @version    CVS: 1.0.0
 * @package    Com_Subscription
 * @author     Rob <pr@floridarangers.com>
 * @copyright  2024 Rob
 * @license    GNU General Public License version 2 or later; see LICENSE.txt
 */

namespace Sub\Component\Subscription\Site\Model;
// No direct access.
defined('_JEXEC') or die;

use \Joomla\CMS\Factory;
use \Joomla\Utilities\ArrayHelper;
use \Joomla\CMS\Language\Text;
use \Joomla\CMS\Table\Table;
use \Joomla\CMS\MVC\Model\ItemModel;
use \Joomla\CMS\Helper\TagsHelper;
use \Joomla\CMS\Object\CMSObject;
use \Joomla\CMS\User\UserFactoryInterface;
use \Sub\Component\Subscription\Site\Helper\SubscriptionHelper;

/**
 * Subscription model.
 *
 * @since  1.0.0
 */
class CheckoutModel extends ItemModel
{
	protected function populateState()
	{
		$app  = Factory::getApplication('com_subscription');
		$params       = $app->getParams();
		$params_array = $params->toArray();
		$this->setState('params', $params);
	}

	public function getItem($id = null)
	{
		if ($this->_item === null)
		{
			$this->_item = false;
	
			$id = Factory::getApplication()->getUserState('com_subscription.edit.product.id');
	
			if (empty($id))
			{
				$id = $this->getState('product.id');
				
				
			}
	
			// Get a level row instance.
			$table = $this->getTable();
	
			// Attempt to load the row.
			if ($table && $table->load($id))
			{
				
	
				// Check published state.
				if ($published = $this->getState('filter.published'))
				{
					if (isset($table->state) && $table->state != $published)
					{
						throw new \Exception(Text::_('COM_SUBSCRIPTION_ITEM_NOT_LOADED'), 403);
					}
				}
	
				// Convert the Table to a clean CMSObject.
				$properties  = $table->getProperties(1);
				$this->_item = ArrayHelper::toObject($properties, CMSObject::class);
	
	
	if ($this->_item->test_mode==1) { 
			
		$this->_item->stripe_product_id = $this->_item->stripe_test_product_id;
			
	}
				
			}
	
			if (empty($this->_item))
			{
				throw new \Exception(Text::_('COM_SUBSCRIPTION_ITEM_NOT_LOADED'), 404);
			}
		}
	
		
	
		 $container = \Joomla\CMS\Factory::getContainer();
	
		 $userFactory = $container->get(UserFactoryInterface::class);
	
		if (isset($this->_item->created_by))
		{
			$user = $userFactory->loadUserById($this->_item->created_by);
			$this->_item->created_by_name = $user->name;
		}
	
		 $container = \Joomla\CMS\Factory::getContainer();
	
		 $userFactory = $container->get(UserFactoryInterface::class);
	
		if (isset($this->_item->modified_by))
		{
			$user = $userFactory->loadUserById($this->_item->modified_by);
			$this->_item->modified_by_name = $user->name;
		}
	
		return $this->_item;
	}
	
	
	
	/**
	 * Get an instance of Table class
	 *
	 * @param   string $type   Name of the Table class to get an instance of.
	 * @param   string $prefix Prefix for the table class name. Optional.
	 * @param   array  $config Array of configuration values for the Table object. Optional.
	 *
	 * @return  Table|bool Table if success, false on failure.
	 */
	public function getTable($type = 'Product', $prefix = 'Administrator', $config = array())
	{
		return parent::getTable($type, $prefix, $config);
	}
	
	/**
	 * Get the id of an item by alias
	 * @param   string $alias Item alias
	 *
	 * @return  mixed
	 * 
	 * @deprecated  No replacement
	 */
	public function getItemIdByAlias($alias)
	{
		$table      = $this->getTable();
		$properties = $table->getProperties();
		$result     = null;
		$aliasKey   = null;
		if (method_exists($this, 'getAliasFieldNameByView'))
		{
			$aliasKey   = $this->getAliasFieldNameByView('product');
		}
		
	
		if (key_exists('alias', $properties))
		{
			$table->load(array('alias' => $alias));
			$result = $table->id;
		}
		elseif (isset($aliasKey) && key_exists($aliasKey, $properties))
		{
			$table->load(array($aliasKey => $alias));
			$result = $table->id;
		}
		
			return $result;
		
	}

}
